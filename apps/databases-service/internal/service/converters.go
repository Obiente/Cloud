package databases

import (
	"encoding/json"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func dbDatabaseToProto(db *database.DatabaseInstance) *databasesv1.DatabaseInstance {
	proto := &databasesv1.DatabaseInstance{
		Id:             db.ID,
		Name:           db.Name,
		Description:    db.Description,
		Status:         databasesv1.DatabaseStatus(db.Status),
		Type:           databasesv1.DatabaseType(db.Type),
		Version:        db.Version,
		Size:           db.Size,
		CpuCores:       db.CPUCores,
		MemoryBytes:    db.MemoryBytes,
		DiskBytes:      db.DiskBytes,
		DiskUsedBytes:  db.DiskUsedBytes,
		MaxConnections: db.MaxConnections,
		Host:           db.Host,
		Port:           db.Port,
		InstanceId:     db.InstanceID,
		NodeId:         db.NodeID,
		OrganizationId: db.OrganizationID,
		CreatedBy:      db.CreatedBy,
		CreatedAt:      timestamppb.New(db.CreatedAt),
		UpdatedAt:      timestamppb.New(db.UpdatedAt),
	}

	if db.LastStartedAt != nil {
		proto.LastStartedAt = timestamppb.New(*db.LastStartedAt)
	}
	if db.DeletedAt != nil {
		proto.DeletedAt = timestamppb.New(*db.DeletedAt)
	}

	// Parse metadata
	if db.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(db.Metadata), &metadata); err == nil {
			proto.Metadata = metadata
		}
	}

	return proto
}

func dbConnectionToProto(conn *database.DatabaseConnection, databaseID string) *databasesv1.DatabaseConnectionInfo {
	proto := &databasesv1.DatabaseConnectionInfo{
		DatabaseId:     databaseID,
		Host:           conn.Host,
		Port:           conn.Port,
		DatabaseName:   conn.DatabaseName,
		Username:       conn.Username,
		Password:       conn.Password, // Only returned on creation/reset
		SslRequired:    conn.SSLRequired,
		SslCertificate: conn.SSLCertificate,
	}

	// Generate connection strings based on database type
	// This would need to be determined from the database instance type
	// For now, we'll generate a generic PostgreSQL URL
	proto.PostgresqlUrl = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=require",
		conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
	proto.MysqlUrl = fmt.Sprintf("mysql://%s:%s@%s:%d/%s?ssl-mode=REQUIRED",
		conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
	proto.MongodbUrl = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?ssl=true",
		conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
	proto.RedisUrl = fmt.Sprintf("redis://:%s@%s:%d",
		conn.Password, conn.Host, conn.Port)

	proto.ConnectionInstructions = fmt.Sprintf(
		"Connect to your database using:\nHost: %s\nPort: %d\nDatabase: %s\nUsername: %s\nPassword: %s\n\nSSL is required for secure connections.",
		conn.Host, conn.Port, conn.DatabaseName, conn.Username, conn.Password,
	)

	return proto
}

func dbBackupToProto(backup *database.DatabaseBackup) *databasesv1.DatabaseBackup {
	proto := &databasesv1.DatabaseBackup{
		Id:           backup.ID,
		DatabaseId:   backup.DatabaseID,
		Name:         backup.Name,
		Description:  backup.Description,
		SizeBytes:    backup.SizeBytes,
		Status:       databasesv1.DatabaseBackupStatus(backup.Status),
		CreatedAt:    timestamppb.New(backup.CreatedAt),
		ErrorMessage: backup.ErrorMessage,
	}

	if backup.CompletedAt != nil {
		proto.CompletedAt = timestamppb.New(*backup.CompletedAt)
	}

	return proto
}

