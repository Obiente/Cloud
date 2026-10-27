package auth

import (
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"
	"testing"
)

func TestDatabaseCatalogCoversEveryRPC(t *testing.T) {
	RegisterDatabaseServiceProcedures()
	registry := GetPermissionRegistry()
	methods := databasesv1.File_obiente_cloud_databases_v1_database_service_proto.Services().ByName("DatabaseService").Methods()
	for i := 0; i < methods.Len(); i++ {
		path := "/obiente.cloud.databases.v1.DatabaseService/" + string(methods.Get(i).Name())
		registry.mu.RLock()
		mapping := registry.mappings[path]
		registry.mu.RUnlock()
		if mapping == nil || mapping.ResourceType != "database" || mapping.Public || mapping.SuperadminOnly {
			t.Errorf("missing assignable database permission for %s", path)
		}
	}
	for _, permission := range []string{PermissionDatabaseRead, PermissionDatabaseCreate, PermissionDatabaseUpdate, PermissionDatabaseDelete, PermissionDatabaseStart, PermissionDatabaseStop, PermissionDatabaseRestart, PermissionDatabaseManage} {
		if registry.IsSuperadminOnly(permission) {
			t.Errorf("organization permission hidden: %s", permission)
		}
		found := false
		for _, definition := range registry.GetAllPermissions(true) {
			if definition.Permission == permission {
				found = true
			}
		}
		if !found {
			t.Errorf("permission absent from role editor catalog: %s", permission)
		}
	}
}
