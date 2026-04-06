package superadmin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ─── Organization moderation ─────────────────────────────────────────────────

func (s *Service) SuspendOrganization(ctx context.Context, req *connect.Request[superadminv1.SuspendOrganizationRequest]) (*connect.Response[superadminv1.SuspendOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.organizations.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	var org database.Organization
	if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load organization: %w", err))
	}

	now := time.Now()
	reason := req.Msg.GetReason()
	var expires *time.Time
	if req.Msg.ExpiresAt != nil {
		t := req.Msg.ExpiresAt.AsTime()
		expires = &t
	}

	updates := map[string]interface{}{
		"status":             "suspended",
		"suspended_at":       now,
		"suspended_by":       user.Id,
		"suspension_reason":  nullableString(reason),
		"suspension_expires": expires,
	}
	if err := database.DB.Model(&org).Updates(updates).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to suspend organization: %w", err))
	}

	logger.Info("[Moderation] Organization %s suspended by %s (reason: %s)", orgID, user.Id, reason)

	return connect.NewResponse(&superadminv1.SuspendOrganizationResponse{
		Message: "Organization suspended successfully",
		Status:  "suspended",
	}), nil
}

func (s *Service) UnsuspendOrganization(ctx context.Context, req *connect.Request[superadminv1.UnsuspendOrganizationRequest]) (*connect.Response[superadminv1.UnsuspendOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.organizations.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	updates := map[string]interface{}{
		"status":             "active",
		"suspended_at":       nil,
		"suspended_by":       nil,
		"suspension_reason":  nil,
		"suspension_expires": nil,
	}
	if err := database.DB.Model(&database.Organization{}).Where("id = ?", orgID).Updates(updates).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unsuspend organization: %w", err))
	}

	logger.Info("[Moderation] Organization %s unsuspended by %s", orgID, user.Id)
	return connect.NewResponse(&superadminv1.UnsuspendOrganizationResponse{Message: "Organization unsuspended successfully"}), nil
}

func (s *Service) BanOrganization(ctx context.Context, req *connect.Request[superadminv1.BanOrganizationRequest]) (*connect.Response[superadminv1.BanOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.organizations.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	now := time.Now()
	reason := req.Msg.GetReason()

	updates := map[string]interface{}{
		"status":     "banned",
		"banned_at":  now,
		"banned_by":  user.Id,
		"ban_reason": nullableString(reason),
		// Also clear any pending suspension
		"suspended_at":       nil,
		"suspended_by":       nil,
		"suspension_reason":  nil,
		"suspension_expires": nil,
	}
	if err := database.DB.Model(&database.Organization{}).Where("id = ?", orgID).Updates(updates).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ban organization: %w", err))
	}

	logger.Info("[Moderation] Organization %s BANNED by %s (reason: %s)", orgID, user.Id, reason)

	return connect.NewResponse(&superadminv1.BanOrganizationResponse{
		Message: "Organization banned",
		Status:  "banned",
	}), nil
}

func (s *Service) UnbanOrganization(ctx context.Context, req *connect.Request[superadminv1.UnbanOrganizationRequest]) (*connect.Response[superadminv1.UnbanOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.organizations.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	updates := map[string]interface{}{
		"status":     "active",
		"banned_at":  nil,
		"banned_by":  nil,
		"ban_reason": nil,
	}
	if err := database.DB.Model(&database.Organization{}).Where("id = ?", orgID).Updates(updates).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unban organization: %w", err))
	}

	logger.Info("[Moderation] Organization %s unbanned by %s", orgID, user.Id)
	return connect.NewResponse(&superadminv1.UnbanOrganizationResponse{Message: "Organization unbanned"}), nil
}

// ─── User ban / suspend ───────────────────────────────────────────────────────

func (s *Service) SuspendUser(ctx context.Context, req *connect.Request[superadminv1.SuspendUserRequest]) (*connect.Response[superadminv1.SuspendUserResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.users.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	targetUserID := req.Msg.GetUserId()
	if targetUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	// Lift any existing active ban for this user first
	if err := liftActiveUserBan(targetUserID, user.Id); err != nil {
		logger.Warn("[Moderation] Failed to lift existing ban for user %s before suspend: %v", targetUserID, err)
	}

	reason := req.Msg.GetReason()
	var expires *time.Time
	if req.Msg.ExpiresAt != nil {
		t := req.Msg.ExpiresAt.AsTime()
		expires = &t
	}

	ban := &database.UserBan{
		ID:        uuid.New().String(),
		UserID:    targetUserID,
		Type:      "suspended",
		BannedBy:  user.Id,
		BannedAt:  time.Now(),
		ExpiresAt: expires,
	}
	if reason != "" {
		ban.Reason = &reason
	}

	if err := database.DB.Create(ban).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create suspension: %w", err))
	}

	logger.Info("[Moderation] User %s suspended by %s", targetUserID, user.Id)

	return connect.NewResponse(&superadminv1.SuspendUserResponse{
		Ban:     userBanToProto(ban),
		Message: "User suspended",
	}), nil
}

func (s *Service) UnsuspendUser(ctx context.Context, req *connect.Request[superadminv1.UnsuspendUserRequest]) (*connect.Response[superadminv1.UnsuspendUserResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.users.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	targetUserID := req.Msg.GetUserId()
	if targetUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	if err := liftActiveUserBan(targetUserID, user.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lift suspension: %w", err))
	}

	logger.Info("[Moderation] User %s unsuspended by %s", targetUserID, user.Id)
	return connect.NewResponse(&superadminv1.UnsuspendUserResponse{Message: "User unsuspended"}), nil
}

func (s *Service) BanUser(ctx context.Context, req *connect.Request[superadminv1.BanUserRequest]) (*connect.Response[superadminv1.BanUserResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.users.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	targetUserID := req.Msg.GetUserId()
	if targetUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	// Lift existing active ban/suspension
	if err := liftActiveUserBan(targetUserID, user.Id); err != nil {
		logger.Warn("[Moderation] Failed to lift existing ban for user %s before ban: %v", targetUserID, err)
	}

	reason := req.Msg.GetReason()
	ban := &database.UserBan{
		ID:       uuid.New().String(),
		UserID:   targetUserID,
		Type:     "banned",
		BannedBy: user.Id,
		BannedAt: time.Now(),
	}
	if reason != "" {
		ban.Reason = &reason
	}

	if err := database.DB.Create(ban).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create ban: %w", err))
	}

	logger.Info("[Moderation] User %s BANNED by %s", targetUserID, user.Id)

	return connect.NewResponse(&superadminv1.BanUserResponse{
		Ban:     userBanToProto(ban),
		Message: "User banned",
	}), nil
}

func (s *Service) UnbanUser(ctx context.Context, req *connect.Request[superadminv1.UnbanUserRequest]) (*connect.Response[superadminv1.UnbanUserResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.users.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	targetUserID := req.Msg.GetUserId()
	if targetUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	if err := liftActiveUserBan(targetUserID, user.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unban user: %w", err))
	}

	logger.Info("[Moderation] User %s unbanned by %s", targetUserID, user.Id)
	return connect.NewResponse(&superadminv1.UnbanUserResponse{Message: "User unbanned"}), nil
}

func (s *Service) GetUserBanStatus(ctx context.Context, req *connect.Request[superadminv1.GetUserBanStatusRequest]) (*connect.Response[superadminv1.GetUserBanStatusResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.users.read") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	targetUserID := req.Msg.GetUserId()
	if targetUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	ban, err := getActiveUserBan(targetUserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get ban status: %w", err))
	}

	resp := &superadminv1.GetUserBanStatusResponse{}
	if ban != nil {
		resp.Ban = userBanToProto(ban)
	}
	return connect.NewResponse(resp), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func getActiveUserBan(userID string) (*database.UserBan, error) {
	var ban database.UserBan
	err := database.DB.
		Where("user_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Order("banned_at DESC").
		First(&ban).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ban, err
}

func liftActiveUserBan(userID, revokedBy string) error {
	now := time.Now()
	return database.DB.Model(&database.UserBan{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]interface{}{
			"revoked_at": now,
			"revoked_by": revokedBy,
		}).Error
}

func userBanToProto(b *database.UserBan) *superadminv1.UserBanInfo {
	if b == nil {
		return nil
	}
	info := &superadminv1.UserBanInfo{
		Id:       b.ID,
		UserId:   b.UserID,
		Type:     b.Type,
		BannedBy: b.BannedBy,
		BannedAt: timestamppb.New(b.BannedAt),
		IsActive: b.RevokedAt == nil && (b.ExpiresAt == nil || b.ExpiresAt.After(time.Now())),
	}
	if b.Reason != nil {
		info.Reason = b.Reason
	}
	if b.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*b.ExpiresAt)
	}
	return info
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
