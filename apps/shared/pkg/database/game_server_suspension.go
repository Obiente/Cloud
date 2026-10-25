package database

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

var ErrGameServerSuspended = errors.New("game server is suspended")

// EnsureGameServerNotSuspended fails closed before mutating a suspended resource.
func EnsureGameServerNotSuspended(ctx context.Context, db *gorm.DB, id string) error {
	if db == nil {
		return fmt.Errorf("cannot verify game server suspension: database unavailable")
	}
	var count int64
	if err := db.WithContext(ctx).Model(&GameServerSuspension{}).Where("game_server_id = ? AND lifted_at IS NULL", id).Count(&count).Error; err != nil {
		return fmt.Errorf("cannot verify game server suspension: %w", err)
	}
	if count > 0 {
		return ErrGameServerSuspended
	}
	return nil
}
