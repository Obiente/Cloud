package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type SupportRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewSupportRepository(db *gorm.DB, cache *RedisCache) *SupportRepository {
	return &SupportRepository{
		db:    db,
		cache: cache,
	}
}

func (r *SupportRepository) CreateTicket(ctx context.Context, ticket *SupportTicket) error {
	if err := r.db.WithContext(ctx).Create(ticket).Error; err != nil {
		return err
	}

	// Cache the newly created ticket
	if r.cache != nil {
		cacheKey := fmt.Sprintf("support:ticket:%s", ticket.ID)
		r.cache.Set(ctx, cacheKey, ticket, 3*time.Minute) // Shorter TTL for support tickets (more dynamic)
	}

	return nil
}

func (r *SupportRepository) GetTicketByID(ctx context.Context, id string) (*SupportTicket, error) {
	cacheKey := fmt.Sprintf("support:ticket:%s", id)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var ticket SupportTicket
			if err := json.Unmarshal([]byte(cachedData), &ticket); err == nil {
				return &ticket, nil
			}
		}
	}

	var ticket SupportTicket
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&ticket).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, ticket, 3*time.Minute)
	}

	return &ticket, nil
}

func (r *SupportRepository) UpdateTicket(ctx context.Context, ticket *SupportTicket) error {
	if err := r.db.WithContext(ctx).
		Model(ticket).
		Updates(ticket).Error; err != nil {
		return err
	}

	// Clear and re-populate cache AFTER successful update
	if r.cache != nil {
		cacheKey := fmt.Sprintf("support:ticket:%s", ticket.ID)
		// Fetch the updated ticket to ensure we have all fields
		var updatedTicket SupportTicket
		if err := r.db.WithContext(ctx).Where("id = ?", ticket.ID).First(&updatedTicket).Error; err == nil {
			r.cache.Set(ctx, cacheKey, updatedTicket, 3*time.Minute)
		} else {
			// If fetch fails, at least clear the cache to avoid stale data
			r.cache.Delete(ctx, cacheKey)
		}
	}

	return nil
}

func (r *SupportRepository) CreateComment(ctx context.Context, comment *TicketComment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return err
	}

	// Invalidate ticket cache when comment is added
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("support:ticket:%s", comment.TicketID))
		// Also cache the comment
		cacheKey := fmt.Sprintf("support:comment:%s", comment.ID)
		r.cache.Set(ctx, cacheKey, comment, 3*time.Minute)
	}

	return nil
}

func (r *SupportRepository) GetCommentsByTicketID(ctx context.Context, ticketID string) ([]*TicketComment, error) {
	var comments []*TicketComment
	if err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}

	return comments, nil
}

