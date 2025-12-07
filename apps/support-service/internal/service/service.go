package support

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	supportv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/support/v1"
	supportv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/support/v1/supportv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Service struct {
	supportv1connect.UnimplementedSupportServiceHandler
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// CreateTicket creates a new support ticket
func (s *Service) CreateTicket(ctx context.Context, req *connect.Request[supportv1.CreateTicketRequest]) (*connect.Response[supportv1.CreateTicketResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Generate ticket ID
	ticketID := fmt.Sprintf("ticket-%d", time.Now().Unix())

	// Create ticket in database
	ticket := &database.SupportTicket{
		ID:            ticketID,
		Subject:       req.Msg.GetSubject(),
		Description:   req.Msg.GetDescription(),
		Status:        int32(supportv1.SupportTicketStatus_OPEN),
		Priority:      int32(req.Msg.GetPriority()),
		Category:      int32(req.Msg.GetCategory()),
		CreatedBy:     userInfo.Id,
		OrganizationID: req.Msg.OrganizationId,
	}

	if err := s.db.Create(ticket).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create ticket: %w", err))
	}

	protoTicket := dbTicketToProto(ticket, 0)
	resolveTicketProfiles(ctx, protoTicket)

	res := connect.NewResponse(&supportv1.CreateTicketResponse{
		Ticket: protoTicket,
	})
	return res, nil
}

// ListTickets lists support tickets
func (s *Service) ListTickets(ctx context.Context, req *connect.Request[supportv1.ListTicketsRequest]) (*connect.Response[supportv1.ListTicketsResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has superadmin support permissions (either hardcoded role or via role bindings)
	isSuperAdmin := auth.HasSuperadminPermission(ctx, userInfo, "superadmin.support.read")

	// Build query
	query := s.db.Model(&database.SupportTicket{})

	// Non-superadmins can only see their own tickets
	if !isSuperAdmin {
		query = query.Where("created_by = ?", userInfo.Id)
	}

	// Apply filters
	if req.Msg.Status != nil {
		query = query.Where("status = ?", int32(*req.Msg.Status))
	}
	if req.Msg.Category != nil {
		query = query.Where("category = ?", int32(*req.Msg.Category))
	}
	if req.Msg.Priority != nil {
		query = query.Where("priority = ?", int32(*req.Msg.Priority))
	}
	if req.Msg.OrganizationId != nil && *req.Msg.OrganizationId != "" {
		query = query.Where("organization_id = ?", *req.Msg.OrganizationId)
	}

	// Set page size (default to 50)
	pageSize := 50
	if req.Msg.PageSize != nil && *req.Msg.PageSize > 0 && *req.Msg.PageSize <= 100 {
		pageSize = int(*req.Msg.PageSize)
	}

	// Apply pagination
	query = query.Order("created_at DESC").Limit(pageSize + 1)

	// Execute query
	var tickets []database.SupportTicket
	if err := query.Find(&tickets).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list tickets: %w", err))
	}

	// Check if there's a next page
	var nextPageToken *string
	if len(tickets) > pageSize {
		tickets = tickets[:pageSize]
		lastID := tickets[len(tickets)-1].ID
		nextPageToken = &lastID
	}

	// Get comment counts for each ticket
	ticketIDs := make([]string, len(tickets))
	for i, t := range tickets {
		ticketIDs[i] = t.ID
	}

	var commentCounts []struct {
		TicketID string
		Count    int64
	}
	if len(ticketIDs) > 0 {
		s.db.Model(&database.TicketComment{}).
			Select("ticket_id, COUNT(*) as count").
			Where("ticket_id IN ?", ticketIDs).
			Group("ticket_id").
			Find(&commentCounts)
	}

	commentCountMap := make(map[string]int32)
	for _, cc := range commentCounts {
		commentCountMap[cc.TicketID] = int32(cc.Count)
	}

	// Convert to proto
	protoTickets := make([]*supportv1.SupportTicket, len(tickets))
	for i, ticket := range tickets {
		count := commentCountMap[ticket.ID]
		protoTickets[i] = dbTicketToProto(&ticket, count)
		resolveTicketProfiles(ctx, protoTickets[i])
	}

	res := connect.NewResponse(&supportv1.ListTicketsResponse{
		Tickets:       protoTickets,
		NextPageToken: nextPageToken,
	})
	return res, nil
}

// GetTicket retrieves a specific ticket by ID
func (s *Service) GetTicket(ctx context.Context, req *connect.Request[supportv1.GetTicketRequest]) (*connect.Response[supportv1.GetTicketResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has superadmin support permissions (either hardcoded role or via role bindings)
	isSuperAdmin := auth.HasSuperadminPermission(ctx, userInfo, "superadmin.support.read")

	// Get ticket
	var ticket database.SupportTicket
	if err := s.db.Where("id = ?", req.Msg.GetTicketId()).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get ticket: %w", err))
	}

	// Check permissions: non-superadmins can only see their own tickets
	if !isSuperAdmin && ticket.CreatedBy != userInfo.Id {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	// Get comment count
	var commentCount int64
	s.db.Model(&database.TicketComment{}).
		Where("ticket_id = ?", ticket.ID).
		Count(&commentCount)

	protoTicket := dbTicketToProto(&ticket, int32(commentCount))
	resolveTicketProfiles(ctx, protoTicket)

	res := connect.NewResponse(&supportv1.GetTicketResponse{
		Ticket: protoTicket,
	})
	return res, nil
}

// UpdateTicket updates a ticket (status, priority, assignee)
func (s *Service) UpdateTicket(ctx context.Context, req *connect.Request[supportv1.UpdateTicketRequest]) (*connect.Response[supportv1.UpdateTicketResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has superadmin support permissions (either hardcoded role or via role bindings)
	isSuperAdmin := auth.HasSuperadminPermission(ctx, userInfo, "superadmin.support.update")

	if !isSuperAdmin {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("only superadmins can update tickets"))
	}

	// Get ticket
	var ticket database.SupportTicket
	if err := s.db.Where("id = ?", req.Msg.GetTicketId()).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get ticket: %w", err))
	}

	// Update fields
	updates := make(map[string]interface{})
	if req.Msg.Status != nil {
		updates["status"] = int32(*req.Msg.Status)
		// Set resolved_at if status is RESOLVED or CLOSED
		if *req.Msg.Status == supportv1.SupportTicketStatus_RESOLVED || *req.Msg.Status == supportv1.SupportTicketStatus_CLOSED {
			now := time.Now()
			updates["resolved_at"] = &now
		} else if ticket.ResolvedAt != nil {
			// Clear resolved_at if status changes away from resolved/closed
			updates["resolved_at"] = nil
		}
	}
	if req.Msg.Priority != nil {
		updates["priority"] = int32(*req.Msg.Priority)
	}
	if req.Msg.AssignedTo != nil {
		updates["assigned_to"] = *req.Msg.AssignedTo
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.db.Model(&ticket).Updates(updates).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update ticket: %w", err))
		}
	}

	// Refresh ticket
	if err := s.db.Where("id = ?", req.Msg.GetTicketId()).First(&ticket).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to refresh ticket: %w", err))
	}

	// Get comment count
	var commentCount int64
	s.db.Model(&database.TicketComment{}).
		Where("ticket_id = ?", ticket.ID).
		Count(&commentCount)

	protoTicket := dbTicketToProto(&ticket, int32(commentCount))
	resolveTicketProfiles(ctx, protoTicket)

	res := connect.NewResponse(&supportv1.UpdateTicketResponse{
		Ticket: protoTicket,
	})
	return res, nil
}

// AddComment adds a comment/reply to a ticket
func (s *Service) AddComment(ctx context.Context, req *connect.Request[supportv1.AddCommentRequest]) (*connect.Response[supportv1.AddCommentResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has superadmin support permissions (either hardcoded role or via role bindings)
	isSuperAdmin := auth.HasSuperadminPermission(ctx, userInfo, "superadmin.support.update")

	// Get ticket to verify it exists and check permissions
	var ticket database.SupportTicket
	if err := s.db.Where("id = ?", req.Msg.GetTicketId()).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get ticket: %w", err))
	}

	// Check permissions: non-superadmins can only comment on their own tickets
	if !isSuperAdmin && ticket.CreatedBy != userInfo.Id {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	// Only superadmins can create internal comments
	isInternal := false
	if req.Msg.Internal != nil && *req.Msg.Internal {
		if !isSuperAdmin {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("only superadmins can create internal comments"))
		}
		isInternal = true
	}

	// Generate comment ID
	commentID := fmt.Sprintf("comment-%d", time.Now().Unix())

	// Create comment
	comment := &database.TicketComment{
		ID:        commentID,
		TicketID:  req.Msg.GetTicketId(),
		Content:   req.Msg.GetContent(),
		CreatedBy: userInfo.Id,
		Internal:  isInternal,
	}

	if err := s.db.Create(comment).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create comment: %w", err))
	}

	// Update ticket's updated_at timestamp
	s.db.Model(&ticket).Update("updated_at", time.Now())

	protoComment := dbCommentToProto(comment)
	
	// Resolve user information if resolver is available
	resolver := organizations.GetUserProfileResolver()
	if resolver != nil && resolver.IsConfigured() {
		if userProfile, err := resolver.Resolve(ctx, comment.CreatedBy); err == nil && userProfile != nil {
			if userProfile.Name != "" {
				protoComment.CreatedByName = &userProfile.Name
			}
			if userProfile.Email != "" {
				protoComment.CreatedByEmail = &userProfile.Email
				// Check if user is superadmin based on email
				testUser := &authv1.User{Email: userProfile.Email, Roles: []string{}}
				protoComment.IsSuperadmin = auth.HasRole(testUser, auth.RoleSuperAdmin)
			}
		}
	}

	res := connect.NewResponse(&supportv1.AddCommentResponse{
		Comment: protoComment,
	})
	return res, nil
}

// ListComments lists comments for a ticket
func (s *Service) ListComments(ctx context.Context, req *connect.Request[supportv1.ListCommentsRequest]) (*connect.Response[supportv1.ListCommentsResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has superadmin support permissions (either hardcoded role or via role bindings)
	isSuperAdmin := auth.HasSuperadminPermission(ctx, userInfo, "superadmin.support.read")

	// Get ticket to verify it exists and check permissions
	var ticket database.SupportTicket
	if err := s.db.Where("id = ?", req.Msg.GetTicketId()).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get ticket: %w", err))
	}

	// Check permissions: non-superadmins can only see their own tickets
	if !isSuperAdmin && ticket.CreatedBy != userInfo.Id {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	// Build query
	query := s.db.Model(&database.TicketComment{}).
		Where("ticket_id = ?", req.Msg.GetTicketId()).
		Order("created_at ASC")

	// Non-superadmins can't see internal comments
	if !isSuperAdmin {
		query = query.Where("internal = ?", false)
	}

	// Execute query
	var comments []database.TicketComment
	if err := query.Find(&comments).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list comments: %w", err))
	}

	// Convert to proto with user information
	protoComments := make([]*supportv1.TicketComment, len(comments))
	resolver := organizations.GetUserProfileResolver()
	for i, comment := range comments {
		protoComment := dbCommentToProto(&comment)
		
		// Resolve user information if resolver is available
		if resolver != nil && resolver.IsConfigured() {
			if userProfile, err := resolver.Resolve(ctx, comment.CreatedBy); err == nil && userProfile != nil {
				if userProfile.Name != "" {
					protoComment.CreatedByName = &userProfile.Name
				}
				if userProfile.Email != "" {
					protoComment.CreatedByEmail = &userProfile.Email
					// Check if user is superadmin based on email
					testUser := &authv1.User{Email: userProfile.Email, Roles: []string{}}
					protoComment.IsSuperadmin = auth.HasRole(testUser, auth.RoleSuperAdmin)
				}
			}
		}
		
		protoComments[i] = protoComment
	}

	res := connect.NewResponse(&supportv1.ListCommentsResponse{
		Comments: protoComments,
	})
	return res, nil
}

// Helper functions for conversion

func dbTicketToProto(dbTicket *database.SupportTicket, commentCount int32) *supportv1.SupportTicket {
	ticket := &supportv1.SupportTicket{
		Id:           dbTicket.ID,
		Subject:      dbTicket.Subject,
		Description:  dbTicket.Description,
		Status:       supportv1.SupportTicketStatus(dbTicket.Status),
		Priority:     supportv1.SupportTicketPriority(dbTicket.Priority),
		Category:     supportv1.SupportTicketCategory(dbTicket.Category),
		CreatedBy:    dbTicket.CreatedBy,
		OrganizationId: dbTicket.OrganizationID,
		CreatedAt:    timestamppb.New(dbTicket.CreatedAt),
		UpdatedAt:    timestamppb.New(dbTicket.UpdatedAt),
		CommentCount: commentCount,
	}

	if dbTicket.AssignedTo != nil {
		ticket.AssignedTo = dbTicket.AssignedTo
	}
	if dbTicket.ResolvedAt != nil {
		ticket.ResolvedAt = timestamppb.New(*dbTicket.ResolvedAt)
	}

	return ticket
}

// resolveTicketProfiles resolves user profile information for a ticket
func resolveTicketProfiles(ctx context.Context, ticket *supportv1.SupportTicket) {
	resolver := organizations.GetUserProfileResolver()
	if resolver == nil || !resolver.IsConfigured() {
		return
	}

	// Resolve created_by profile
	if ticket.CreatedBy != "" {
		if userProfile, err := resolver.Resolve(ctx, ticket.CreatedBy); err == nil && userProfile != nil {
			if userProfile.Name != "" {
				ticket.CreatedByName = &userProfile.Name
			}
			if userProfile.Email != "" {
				ticket.CreatedByEmail = &userProfile.Email
			}
		}
	}

	// Resolve assigned_to profile
	if ticket.AssignedTo != nil && *ticket.AssignedTo != "" {
		if userProfile, err := resolver.Resolve(ctx, *ticket.AssignedTo); err == nil && userProfile != nil {
			if userProfile.Name != "" {
				ticket.AssignedToName = &userProfile.Name
			}
			if userProfile.Email != "" {
				ticket.AssignedToEmail = &userProfile.Email
			}
		}
	}
}

func dbCommentToProto(dbComment *database.TicketComment) *supportv1.TicketComment {
	return &supportv1.TicketComment{
		Id:        dbComment.ID,
		TicketId:  dbComment.TicketID,
		Content:   dbComment.Content,
		CreatedBy: dbComment.CreatedBy,
		Internal:  dbComment.Internal,
		CreatedAt: timestamppb.New(dbComment.CreatedAt),
		UpdatedAt: timestamppb.New(dbComment.UpdatedAt),
	}
}

