package orchestrator

import (
	"context"
	"encoding/json"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuspensionStopsVMAndReconciliationStopsExternalRestart(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	prior := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prior; raw.Close() })
	if err := db.AutoMigrate(&database.VPSInstance{}, &database.VPSSuspension{}, &database.Organization{}); err != nil {
		t.Fatal(err)
	}
	node, instance := "test-node", "123"
	ctx := context.Background()
	for _, row := range []interface{}{&database.Organization{ID: "org", Status: "active"}, &database.VPSInstance{ID: "vps-test", OrganizationID: "org", NodeID: &node, InstanceID: &instance, Status: int32(vpsv1.VPSStatus_RUNNING)}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	status := "running"
	stops := 0
	failed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/status/stop") {
			if failed {
				http.Error(w, "outage", 503)
				return
			}
			stops++
			status = "stopped"
			w.Write([]byte(`{"data":"test-task"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/status/current") {
			json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]string{"status": status}})
			return
		}
		t.Errorf("unexpected hypervisor operation: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", 500)
	}))
	defer server.Close()
	client, err := NewProxmoxClient(&ProxmoxConfig{APIURL: server.URL, TokenID: "test-token", Secret: "test-only"})
	if err != nil {
		t.Fatal(err)
	}
	manager := &VPSManager{}
	manager.proxmoxClients.Store(node, client)
	if err := database.SuspendVPS(ctx, db, "vps-test", "test-admin", "test"); err != nil {
		t.Fatal(err)
	}
	if err := manager.StopVPS(ctx, "vps-test", true); err != nil {
		t.Fatal(err)
	}
	if stops != 1 || status != "stopped" {
		t.Fatal("suspension did not stop VM")
	}
	status = "running"
	failed = true
	if err := manager.SyncVPSStatusFromProxmox(ctx, "vps-test"); err == nil {
		t.Fatal("failed stop reported success")
	}
	if err := manager.StartVPS(ctx, "vps-test"); err == nil {
		t.Fatal("suspension allowed start")
	}
	failed = false
	if err := manager.SyncVPSStatusFromProxmox(ctx, "vps-test"); err != nil {
		t.Fatal(err)
	}
	if stops != 2 || status != "stopped" {
		t.Fatal("reconciliation did not recover failed suspension")
	}
	var vps database.VPSInstance
	if err := db.First(&vps, "id = ?", "vps-test").Error; err != nil {
		t.Fatal(err)
	}
	if vps.Status != int32(vpsv1.VPSStatus_SUSPENDED) {
		t.Fatal("observation cleared restriction")
	}
}

func TestDeleteVMRequiresConfirmedOwnershipAndState(t *testing.T) {
	for _, tc := range []struct {
		name       string
		code       int
		body       string
		statusCode int
		allowed    bool
	}{
		{"unavailable", 503, `{}`, 200, false}, {"malformed", 200, `broken`, 200, false}, {"wrong-owner", 200, `{"data":{"name":"another-vps"}}`, 200, false},
		{"missing-owner", 200, `{"data":{}}`, 200, false}, {"state-unavailable", 200, `{"data":{"name":"vps-test"}}`, 503, false}, {"verified", 200, `{"data":{"name":"vps-test"}}`, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutations := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" {
					mutations++
					w.Write([]byte(`{"data":"test-task"}`))
					return
				}
				if strings.HasSuffix(r.URL.Path, "/config") {
					w.WriteHeader(tc.code)
					w.Write([]byte(tc.body))
					return
				}
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(`{"data":{"status":"stopped"}}`))
			}))
			defer server.Close()
			client, err := NewProxmoxClient(&ProxmoxConfig{APIURL: server.URL, TokenID: "test-token", Secret: "test-only"})
			if err != nil {
				t.Fatal(err)
			}
			err = client.DeleteVM(context.Background(), "test-node", 123, "vps-test")
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed %v, error %v", tc.allowed, err)
			}
			if !tc.allowed && mutations != 0 {
				t.Fatal("ownership failure caused a destructive request")
			}
			if tc.allowed && mutations != 1 {
				t.Fatal("verified deletion did not execute exactly once")
			}
		})
	}
}
