package databases

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func dbDatabaseToProto(db *database.DatabaseInstance) *databasesv1.DatabaseInstance {
	host := db.Host
	if canonicalHost := database.DefaultMyObienteCloudDomain(db.ID); canonicalHost != "" {
		host = &canonicalHost
	}

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
		Host:           host,
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

	if db.AutoSleepSeconds > 0 {
		proto.AutoSleepSeconds = &db.AutoSleepSeconds
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
	return buildDatabaseConnectionInfo(databasesv1.DatabaseType_DATABASE_TYPE_UNSPECIFIED, conn, databaseID, "")
}

func buildDatabaseConnectionInfo(
	dbType databasesv1.DatabaseType,
	conn *database.DatabaseConnection,
	databaseID string,
	password string,
) *databasesv1.DatabaseConnectionInfo {
	host := conn.Host
	if canonicalHost := database.DefaultMyObienteCloudDomain(databaseID); canonicalHost != "" {
		host = canonicalHost
	}

	proto := &databasesv1.DatabaseConnectionInfo{
		DatabaseId:     databaseID,
		Host:           host,
		Port:           conn.Port,
		DatabaseName:   conn.DatabaseName,
		Username:       conn.Username,
		SslRequired:    conn.SSLRequired,
		SslCertificate: conn.SSLCertificate,
	}
	if password != "" {
		proto.Password = password
	}

	escapedUser := url.QueryEscape(conn.Username)
	escapedPassword := url.QueryEscape(password)
	authSegment := escapedUser
	if password != "" {
		authSegment = fmt.Sprintf("%s:%s", escapedUser, escapedPassword)
	}
	redisAuthSegment := ""
	if password != "" {
		redisAuthSegment = ":" + escapedPassword + "@"
	}

	proto.PostgresqlUrl = fmt.Sprintf("postgresql://%s@%s:%d/%s?sslmode=require",
		authSegment, host, conn.Port, conn.DatabaseName)
	proto.MysqlUrl = fmt.Sprintf("mysql://%s@%s:%d/%s?ssl-mode=REQUIRED",
		authSegment, host, conn.Port, conn.DatabaseName)
	proto.MongodbUrl = fmt.Sprintf("mongodb://%s@%s:%d/%s?ssl=true",
		authSegment, host, conn.Port, conn.DatabaseName)
	proto.RedisUrl = fmt.Sprintf("redis://%s%s:%d",
		redisAuthSegment, host, conn.Port)

	proto.ConnectionInstructions = fmt.Sprintf(
		"Connect to your database using:\nHost: %s\nPort: %d\nDatabase: %s\nUsername: %s\n\nSSL is required for secure connections.",
		host, conn.Port, conn.DatabaseName, conn.Username,
	)

	if dbType == databasesv1.DatabaseType_REDIS {
		proto.ConnectionInstructions = fmt.Sprintf(
			"Connect to your Redis database using:\nHost: %s\nPort: %d\nPassword: available in this response only.\n\nUse TLS-aware clients if you expose Redis over a secure endpoint.",
			host, conn.Port,
		)
	}

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
