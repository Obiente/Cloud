package database

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"database/sql"
	"database/sql/driver"
	"errors"
	"gorm.io/gorm"
)

type runtimeLockKey struct{}
type runtimeLock struct {
	id     string
	active atomic.Bool
}

var localRuntimeLocks sync.Map

// Serialize recovery with explicit lifecycle requests, including across service
// replicas. Nested synchronous manager operations share the same lock.
func WithResourceRuntimeLock(ctx context.Context, id string, operation func(context.Context) error) error {
	if held, ok := ctx.Value(runtimeLockKey{}).(*runtimeLock); ok && held.id == id && held.active.Load() {
		return operation(ctx)
	}
	run := func() error {
		held := &runtimeLock{id: id}
		held.active.Store(true)
		defer held.active.Store(false)
		return operation(context.WithValue(ctx, runtimeLockKey{}, held))
	}
	if DB == nil || DB.Dialector.Name() != "postgres" {
		value, _ := localRuntimeLocks.LoadOrStore(id, make(chan struct{}, 1))
		lock := value.(chan struct{})
		select {
		case lock <- struct{}{}:
			defer func() { <-lock }()
			return run()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte("obiente-resource-runtime:" + id))
	key := int64(hash.Sum64())
	return DB.WithContext(ctx).Connection(func(conn *gorm.DB) (retErr error) {
		for {
			var acquired bool
			if err := conn.Raw("SELECT pg_try_advisory_lock(?)", key).Scan(&acquired).Error; err != nil {
				return err
			}
			if acquired {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := conn.WithContext(cleanup).Exec("SELECT pg_advisory_unlock(?)", key).Error; err != nil {
				if pinned, ok := conn.Statement.ConnPool.(*sql.Conn); ok {
					_ = pinned.Raw(func(any) error { return driver.ErrBadConn })
				}
				retErr = errors.Join(retErr, fmt.Errorf("release resource runtime lock: %w", err))
			}
		}()
		return run()
	})
}
