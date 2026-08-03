package deployments

import (
	"context"
	"sync"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/gorm"
)

var localServiceLocks sync.Map

// withDistributedLock serializes a short state transition across service
// replicas. PostgreSQL transaction advisory locks are released automatically
// on commit, rollback, or connection loss. Tests and local non-PostgreSQL
// databases use the same keyed locking semantics within one process.
func withDistributedLock(ctx context.Context, key string, fn func() error) error {
	if database.DB.Dialector.Name() == "postgres" {
		return database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error; err != nil {
				return err
			}
			return fn()
		})
	}

	value, _ := localServiceLocks.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()
	return fn()
}
