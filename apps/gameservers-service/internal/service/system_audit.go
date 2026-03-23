package gameservers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
)

const (
	systemAuditTimeout           = 5 * time.Second
	gameServerAuditServiceName   = "GameServerService"
	gameServerAuditResourceType  = "game_server"
	gameServerAuditIPAddress     = "internal"
	gameServerAuditSourceMonitor = "health_monitor"
)

func (s *Service) createSystemGameServerAuditLog(gameServer *database.GameServer, gameServerID string, action string, reason string, source string, responseStatus int32, fields map[string]interface{}, actionErr error) {
	if gameServer == nil && gameServerID == "" {
		return
	}

	auditCtx, cancel := context.WithTimeout(s.createSystemContext(), systemAuditTimeout)
	defer cancel()

	if gameServer == nil {
		loadedGameServer, err := s.repo.GetByID(auditCtx, gameServerID)
		if err != nil {
			logger.Warn("[GameServerAudit] Failed to load game server %s for audit log: %v", gameServerID, err)
		} else {
			gameServer = loadedGameServer
		}
	}

	resourceID := gameServerID
	if resourceID == "" && gameServer != nil {
		resourceID = gameServer.ID
	}
	if resourceID == "" {
		return
	}

	requestDataMap := map[string]interface{}{
		"reason": reason,
		"source": source,
	}
	if gameServer != nil && gameServer.Name != "" {
		requestDataMap["gameServerName"] = gameServer.Name
	}
	for key, value := range fields {
		requestDataMap[key] = value
	}

	requestDataBytes, err := json.Marshal(requestDataMap)
	if err != nil {
		logger.Warn("[GameServerAudit] Failed to marshal audit request data for game server %s: %v", resourceID, err)
		requestDataBytes = []byte("{}")
	}

	var organizationID *string
	if gameServer != nil && gameServer.OrganizationID != "" {
		organizationID = &gameServer.OrganizationID
	}

	var errorMessage *string
	if actionErr != nil {
		msg := actionErr.Error()
		errorMessage = &msg
	}

	resourceType := gameServerAuditResourceType
	userAgent := "gameservers-service/" + source

	if err := middleware.CreateAuditLog(auditCtx, middleware.AuditEntry{
		UserID:         "system",
		OrganizationID: organizationID,
		Action:         action,
		Service:        gameServerAuditServiceName,
		ResourceType:   &resourceType,
		ResourceID:     &resourceID,
		IPAddress:      gameServerAuditIPAddress,
		UserAgent:      userAgent,
		RequestData:    string(requestDataBytes),
		ResponseStatus: responseStatus,
		ErrorMessage:   errorMessage,
		DurationMs:     0,
	}); err != nil {
		logger.Warn("[GameServerAudit] Failed to write audit log for game server %s action %s: %v", resourceID, action, err)
	}
}
