package auth

import (
	"testing"
	"time"
)

func TestPermissionCatalogCompletesWithConcurrentRegistration(t *testing.T) {
	r := &PermissionRegistry{mappings: map[string]*PermissionMapping{}, byPerm: map[string][]string{}, public: map[string]bool{}}
	r.RegisterProcedure("/example/read", "database.read", "database", "read", "Read databases", false)
	done := make(chan struct{}, 2)
	go func() {
		for range 2000 {
			r.GetAllPermissions(true)
		}
		done <- struct{}{}
	}()
	go func() {
		for range 2000 {
			r.RegisterProcedure("/example/read", "database.read", "database", "read", "Read databases", false)
		}
		done <- struct{}{}
	}()
	for range 2 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("permission catalog deadlocked with registration")
		}
	}
}
