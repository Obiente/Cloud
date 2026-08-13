package main

import (
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizePreviewACMEChallengeCNAME(t *testing.T) {
	t.Parallel()

	got, err := normalizePreviewACMEChallengeCNAME(" _ACME-PREVIEW.Example.NET ")
	if err != nil {
		t.Fatalf("normalize challenge CNAME: %v", err)
	}
	if got != "_acme-preview.example.net." {
		t.Fatalf("normalized CNAME = %q", got)
	}

	for _, value := range []string{
		"_acme-challenge.my.obiente.cloud",
		"my.obiente.cloud",
		"nested.my.obiente.cloud",
		"invalid name",
		"-invalid.example.net",
		"invalid-.example.net",
		"täst.example.net",
	} {
		_, err := normalizePreviewACMEChallengeCNAME(value)
		if err == nil {
			t.Errorf("invalid CNAME target %q was accepted", value)
		}
	}

	if _, err := normalizePreviewACMEChallengeCNAME(""); err != nil {
		t.Fatalf("empty optional CNAME: %v", err)
	}
}

func TestCollectDatabaseDNSRecordsSkipsUnresolvedOwner(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:dns-pusher-owner-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&database.DatabaseInstance{}, &database.DatabaseLocation{}, &database.NodeMetadata{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	databaseID := "db-dns-pusher-owner-test"
	containerID := "container-dns-pusher-owner-test"
	if err := db.Create(&database.DatabaseInstance{ID: databaseID, InstanceID: &containerID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := db.Create(&database.DatabaseLocation{
		ID:          database.DatabaseLocationID(databaseID, containerID),
		DatabaseID:  databaseID,
		NodeID:      "missing-owner-node",
		NodeIP:      "203.0.113.55",
		ContainerID: containerID,
		Status:      "running",
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("create database location: %v", err)
	}

	records := collectDatabaseDNSRecords(
		[]databaseDNSRow{{DatabaseID: databaseID}},
		map[string][]string{"default": {"192.0.2.10"}},
		60,
	)
	if len(records) != 0 {
		t.Fatalf("expected unresolved database owner to be omitted, got %#v", records)
	}
}

func TestHandlePreviewACMEChallengeReturnsCNAMEForTXTLookup(t *testing.T) {
	t.Parallel()
	server := &DNSServer{previewACMEChallengeCNAME: "_acme-preview.example.net."}
	message := new(dns.Msg)
	question := dns.Question{
		Name:   previewACMEChallengeName,
		Qtype:  dns.TypeTXT,
		Qclass: dns.ClassINET,
	}

	if !server.handlePreviewACMEChallenge(message, question) {
		t.Fatal("preview ACME challenge was not handled")
	}
	if len(message.Answer) != 1 {
		t.Fatalf("answer count = %d, want 1", len(message.Answer))
	}
	cname, ok := message.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("answer type = %T, want *dns.CNAME", message.Answer[0])
	}
	if cname.Hdr.Name != previewACMEChallengeName || cname.Target != server.previewACMEChallengeCNAME {
		t.Fatalf("unexpected CNAME answer: %#v", cname)
	}
	if !strings.EqualFold(cname.Hdr.Name, question.Name) {
		t.Fatalf("CNAME owner = %q, want %q", cname.Hdr.Name, question.Name)
	}
}

func TestHandlePreviewACMEChallengeIgnoresOtherNamesAndMissingTarget(t *testing.T) {
	t.Parallel()
	message := new(dns.Msg)

	server := &DNSServer{previewACMEChallengeCNAME: "_acme-preview.example.net."}
	if server.handlePreviewACMEChallenge(message, dns.Question{Name: "app.my.obiente.cloud.", Qtype: dns.TypeTXT}) {
		t.Fatal("non-challenge name was handled")
	}

	server.previewACMEChallengeCNAME = ""
	if server.handlePreviewACMEChallenge(message, dns.Question{Name: previewACMEChallengeName, Qtype: dns.TypeTXT}) {
		t.Fatal("challenge without a configured target was handled")
	}
}
