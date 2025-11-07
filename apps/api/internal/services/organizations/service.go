package organizations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/email"
	"api/internal/logger"
	"api/internal/pricing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	defaultConsoleURL        = "https://app.obiente.cloud"
	defaultOwnerFallbackRole = "admin"
)

type Config struct {
	EmailSender  email.Sender
	ConsoleURL   string
	SupportEmail string
}

type Service struct {
	organizationsv1connect.UnimplementedOrganizationServiceHandler
	mailer       email.Sender
	consoleURL   string
	supportEmail string
}

func NewService(cfg Config) organizationsv1connect.OrganizationServiceHandler {
	consoleURL := strings.TrimSuffix(strings.TrimSpace(cfg.ConsoleURL), "/")
	if consoleURL == "" {
		consoleURL = defaultConsoleURL
	}

	return &Service{mailer: cfg.EmailSender, consoleURL: consoleURL, supportEmail: strings.TrimSpace(cfg.SupportEmail)}
}

func (s *Service) ListOrganizations(ctx context.Context, req *connect.Request[organizationsv1.ListOrganizationsRequest]) (*connect.Response[organizationsv1.ListOrganizationsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	type row struct {
		Id, Name, Slug, Plan, Status string
		Domain                       *string
		Credits                      int64
		TotalPaidCents               int64
		CreatedAt                    time.Time
	}

	var rows []row
	onlyMine := req.Msg.GetOnlyMine()
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)

	// If onlyMine is true, or user is not a superadmin, filter to user's memberships
	if onlyMine || !isSuperAdmin {
		ensurePersonalOrg(user.Id)
		if err := database.DB.Raw(`
			SELECT o.id, o.name, o.slug, o.plan, o.status, o.domain, o.credits, o.total_paid_cents, o.created_at
			FROM organizations o
			JOIN organization_members m ON m.organization_id = o.id
			WHERE m.user_id = ?
			ORDER BY o.created_at DESC
		`, user.Id).Scan(&rows).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list organizations: %w", err))
		}
	} else {
		// Superadmin gets all organizations (when onlyMine is false/unset)
		if err := database.DB.Raw(`
			SELECT o.id, o.name, o.slug, o.plan, o.status, o.domain, o.credits, o.total_paid_cents, o.created_at
			FROM organizations o
			ORDER BY o.created_at DESC
		`).Scan(&rows).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list organizations: %w", err))
		}
	}

	orgs := make([]*organizationsv1.Organization, 0, len(rows))
	for _, r := range rows {
		// Ensure organization has a plan assigned (defaults to Starter plan)
		// This ensures plan info is available when converting to proto
		if err := EnsurePlanAssigned(r.Id); err != nil {
			log.Printf("[ListOrganizations] Warning: failed to ensure plan assigned for org %s: %v", r.Id, err)
			// Continue anyway - plan info just won't be populated
		}
		
		// Convert row to database.Organization for organizationToProto
		org := &database.Organization{
			ID:            r.Id,
			Name:          r.Name,
			Slug:          r.Slug,
			Plan:          r.Plan,
			Status:        r.Status,
			Domain:         r.Domain,
			Credits:       r.Credits,
			TotalPaidCents: r.TotalPaidCents,
			CreatedAt:     r.CreatedAt,
		}
		orgProto := organizationToProto(org)
		if orgProto.PlanInfo == nil {
			log.Printf("[ListOrganizations] Warning: PlanInfo is nil for org %s after EnsurePlanAssigned", r.Id)
		} else {
			log.Printf("[ListOrganizations] Successfully loaded plan info for org %s: plan=%s", r.Id, orgProto.PlanInfo.PlanName)
		}
		orgs = append(orgs, orgProto)
	}

	res := connect.NewResponse(&organizationsv1.ListOrganizationsResponse{
		Organizations: orgs,
		Pagination:    &organizationsv1.Pagination{Page: 1, PerPage: int32(len(orgs)), Total: int32(len(orgs)), TotalPages: 1},
	})
	return res, nil
}

func (s *Service) CreateOrganization(ctx context.Context, req *connect.Request[organizationsv1.CreateOrganizationRequest]) (*connect.Response[organizationsv1.CreateOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization name is required"))
	}
	slug := req.Msg.GetSlug()
	if slug == "" {
		slug = normalizeSlug(name)
	}
	plan := req.Msg.GetPlan()
	if plan == "" {
		plan = "starter"
	}
	now := time.Now()
	org := &database.Organization{ID: generateID("org"), Name: name, Slug: slug, Plan: plan, Status: "active", Credits: 0, CreatedAt: now}
	if err := database.DB.Create(org).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create org: %w", err))
	}
	// add creator as owner
	m := &database.OrganizationMember{ID: generateID("mem"), OrganizationID: org.ID, UserID: user.Id, Role: "owner", Status: "active", JoinedAt: now}
	_ = database.DB.Create(m).Error
	// Ensure organization has a plan assigned (defaults to Starter plan)
	_ = EnsurePlanAssigned(org.ID)
	return connect.NewResponse(&organizationsv1.CreateOrganizationResponse{Organization: organizationToProto(org)}), nil
}

func (s *Service) GetOrganization(_ context.Context, req *connect.Request[organizationsv1.GetOrganizationRequest]) (*connect.Response[organizationsv1.GetOrganizationResponse], error) {
	var r database.Organization
	if err := database.DB.First(&r, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}
	// Ensure organization has a plan assigned (defaults to Starter plan)
	if err := EnsurePlanAssigned(r.ID); err != nil {
		log.Printf("[GetOrganization] Warning: failed to ensure plan assigned for org %s: %v", r.ID, err)
		// Continue anyway - plan info just won't be populated
	}
	// Reload organization to get latest data (though organizationToProto will query quota separately)
	return connect.NewResponse(&organizationsv1.GetOrganizationResponse{Organization: organizationToProto(&r)}), nil
}

func (s *Service) UpdateOrganization(ctx context.Context, req *connect.Request[organizationsv1.UpdateOrganizationRequest]) (*connect.Response[organizationsv1.UpdateOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if err := s.authorizeOrgRoles(ctx, req.Msg.GetOrganizationId(), user, "owner", "admin"); err != nil {
		return nil, err
	}

	var org database.Organization
	if err := database.DB.First(&org, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}
	if name := strings.TrimSpace(req.Msg.GetName()); name != "" {
		org.Name = name
	}
	if req.Msg.Domain != nil {
		d := req.Msg.GetDomain()
		if d == "" {
			org.Domain = nil
		} else {
			org.Domain = &d
		}
	}
	if err := database.DB.Save(&org).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update org: %w", err))
	}
	return connect.NewResponse(&organizationsv1.UpdateOrganizationResponse{Organization: organizationToProto(&org)}), nil
}

func (s *Service) ListMembers(ctx context.Context, req *connect.Request[organizationsv1.ListMembersRequest]) (*connect.Response[organizationsv1.ListMembersResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if err := s.authorizeOrgRoles(ctx, req.Msg.GetOrganizationId(), user); err != nil {
		return nil, err
	}

	var members []database.OrganizationMember
	if err := database.DB.Where("organization_id = ?", req.Msg.GetOrganizationId()).Find(&members).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list members: %w", err))
	}

	resolver := getUserProfileResolver()
	list := make([]*organizationsv1.OrganizationMember, 0, len(members))
	for _, member := range members {
		userProto := buildUserProfile(ctx, resolver, member)
		om := &organizationsv1.OrganizationMember{
			Id:       member.ID,
			Role:     member.Role,
			Status:   member.Status,
			JoinedAt: timestamppb.New(member.JoinedAt),
			User:     userProto,
		}
		list = append(list, om)
	}

	res := connect.NewResponse(&organizationsv1.ListMembersResponse{
		Members: list,
		Pagination: &organizationsv1.Pagination{
			Page:       1,
			PerPage:    int32(len(list)),
			Total:      int32(len(list)),
			TotalPages: 1,
		},
	})
	return res, nil
}

func (s *Service) InviteMember(ctx context.Context, req *connect.Request[organizationsv1.InviteMemberRequest]) (*connect.Response[organizationsv1.InviteMemberResponse], error) {
	inviter, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	emailAddr := strings.TrimSpace(req.Msg.GetEmail())
	if emailAddr == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("member email is required"))
	}
	// Normalize email to lowercase for consistency
	emailAddr = strings.ToLower(emailAddr)

	var org database.Organization
	if err := database.DB.First(&org, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}

	if err := s.authorizeOrgRoles(ctx, org.ID, inviter, "owner", "admin"); err != nil {
		return nil, err
	}

	role := req.Msg.GetRole()
	if role == "" {
		role = "member"
	}
	if strings.EqualFold(role, "owner") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner role cannot be invited; transfer flow pending"))
	}

	// Check if user is already invited to this organization (case-insensitive email matching)
	var existingMember database.OrganizationMember
	emailLower := strings.ToLower(emailAddr)
	pendingUserID := "pending:" + emailLower
	err = database.DB.Where("organization_id = ? AND LOWER(user_id) = ? AND status = ?", org.ID, strings.ToLower(pendingUserID), "invited").First(&existingMember).Error
	
	var m *database.OrganizationMember
	if err == nil {
		// User is already invited - update role and resend invite
		existingMember.Role = strings.ToLower(role)
		existingMember.JoinedAt = time.Now() // Update invite timestamp
		if err := database.DB.Save(&existingMember).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update existing invite: %w", err))
		}
		m = &existingMember
		
		// Resend invite email with rate limiting
		// Check rate limit first
		const rateLimitWindow = 5 * time.Minute
		if m.LastInviteSentAt != nil {
			timeSinceLastSend := time.Since(*m.LastInviteSentAt)
			if timeSinceLastSend < rateLimitWindow {
				remainingTime := rateLimitWindow - timeSinceLastSend
				return nil, connect.NewError(
					connect.CodeResourceExhausted,
					fmt.Errorf("invitation was recently sent. please wait %v before resending", remainingTime.Round(time.Second)),
				)
			}
		}
		
		// Send invite email (errors are logged but don't fail the invite update)
		if err := s.dispatchInviteEmail(ctx, &org, m, inviter, emailAddr); err != nil {
			log.Printf("[Organizations] failed to resend invite email for existing member %s: %v", m.ID, err)
			// Continue - member is updated even if email fails
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// New invite - create member
		m = &database.OrganizationMember{ID: generateID("mem"), OrganizationID: org.ID, UserID: pendingUserID, Role: strings.ToLower(role), Status: "invited", JoinedAt: time.Now()}
		if err := database.DB.Create(m).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invite member: %w", err))
		}

		// Send invite email (errors are logged but don't fail the invite creation)
		if err := s.dispatchInviteEmail(ctx, &org, m, inviter, emailAddr); err != nil {
			log.Printf("[Organizations] failed to send invite email for new member %s: %v", m.ID, err)
			// Continue - member is created even if email fails
		}
	} else {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check existing invite: %w", err))
	}

	om := &organizationsv1.OrganizationMember{Id: m.ID, Role: m.Role, Status: m.Status, JoinedAt: timestamppb.New(m.JoinedAt), User: &authv1.User{Id: m.UserID, Email: emailAddr, Name: deriveNameFromEmail(emailAddr)}}
	return connect.NewResponse(&organizationsv1.InviteMemberResponse{Member: om}), nil
}

func (s *Service) ResendInvite(ctx context.Context, req *connect.Request[organizationsv1.ResendInviteRequest]) (*connect.Response[organizationsv1.ResendInviteResponse], error) {
	inviter, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	var member database.OrganizationMember
	if err := database.DB.Where("id = ? AND organization_id = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId()).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("resend invite: %w", err))
	}

	// Only allow resending invites for pending members
	if member.Status != "invited" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("member is not in invited status"))
	}

	// Verify the member is a pending invite (has "pending:" prefix)
	if !strings.HasPrefix(member.UserID, "pending:") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("member is not a pending invite"))
	}

	// Extract email from UserID
	emailAddr := strings.TrimPrefix(member.UserID, "pending:")

	var org database.Organization
	if err := database.DB.First(&org, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}

	// Authorize: user must be owner or admin of the organization, or superadmin
	if err := s.authorizeOrgRoles(ctx, org.ID, inviter, "owner", "admin"); err != nil {
		// Allow superadmins to resend invites for any organization
		if !auth.HasRole(inviter, auth.RoleSuperAdmin) {
			return nil, err
		}
	}

	// Rate limiting: only apply if last invite was successfully sent
	// If email failed previously, allow resending immediately
	const rateLimitWindow = 5 * time.Minute // 5 minutes between resends
	if member.LastInviteSentAt != nil {
		timeSinceLastSend := time.Since(*member.LastInviteSentAt)
		if timeSinceLastSend < rateLimitWindow {
			remainingTime := rateLimitWindow - timeSinceLastSend
			return nil, connect.NewError(
				connect.CodeResourceExhausted,
				fmt.Errorf("please wait %v before resending the invitation", remainingTime.Round(time.Second)),
			)
		}
	}

	// Resend the invite email
	if err := s.dispatchInviteEmail(ctx, &org, &member, inviter, emailAddr); err != nil {
		// If email fails, don't update LastInviteSentAt, so rate limit won't apply
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send invite email: %w", err))
	}

	return connect.NewResponse(&organizationsv1.ResendInviteResponse{Success: true}), nil
}

func (s *Service) ListMyInvites(ctx context.Context, req *connect.Request[organizationsv1.ListMyInvitesRequest]) (*connect.Response[organizationsv1.ListMyInvitesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if user.Email == "" {
		return connect.NewResponse(&organizationsv1.ListMyInvitesResponse{
			Invites:    []*organizationsv1.PendingInvite{},
			Pagination: &organizationsv1.Pagination{Page: 1, PerPage: 0, Total: 0, TotalPages: 0},
		}), nil
	}

	// Find all pending invites for this user's email (case-insensitive)
	pendingUserID := "pending:" + strings.ToLower(user.Email)
	var members []database.OrganizationMember
	if err := database.DB.Where("LOWER(user_id) = ? AND status = ?", strings.ToLower(pendingUserID), "invited").Find(&members).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list invites: %w", err))
	}

	// Get organization details for each invite
	invites := make([]*organizationsv1.PendingInvite, 0, len(members))
	for _, member := range members {
		var org database.Organization
		if err := database.DB.First(&org, "id = ?", member.OrganizationID).Error; err != nil {
			log.Printf("[Organizations] failed to load organization %s for invite: %v", member.OrganizationID, err)
			continue
		}

		invites = append(invites, &organizationsv1.PendingInvite{
			Id:             member.ID,
			OrganizationId: org.ID,
			OrganizationName: org.Name,
			Role:           member.Role,
			InvitedAt:      timestamppb.New(member.JoinedAt),
			InviterEmail:   "", // TODO: Track inviter in future enhancement
		})
	}

	return connect.NewResponse(&organizationsv1.ListMyInvitesResponse{
		Invites: invites,
		Pagination: &organizationsv1.Pagination{
			Page:       1,
			PerPage:    int32(len(invites)),
			Total:      int32(len(invites)),
			TotalPages: 1,
		},
	}), nil
}

func (s *Service) AcceptInvite(ctx context.Context, req *connect.Request[organizationsv1.AcceptInviteRequest]) (*connect.Response[organizationsv1.AcceptInviteResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if user.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user email is required to accept invite"))
	}

	// Find the pending invite
	var member database.OrganizationMember
	if err := database.DB.Where("id = ? AND organization_id = ? AND status = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId(), "invited").First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invite not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("accept invite: %w", err))
	}

	// Verify the invite is for this user's email (case-insensitive)
	pendingUserID := "pending:" + strings.ToLower(user.Email)
	if !strings.EqualFold(member.UserID, pendingUserID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("this invite is not for your email address"))
	}

	// Get organization
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}

	// Update member to active status and set user ID
	member.UserID = user.Id
	member.Status = "active"
	member.JoinedAt = time.Now()
	if err := database.DB.Save(&member).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("accept invite: %w", err))
	}

	// Build response
	om := &organizationsv1.OrganizationMember{
		Id:       member.ID,
		Role:     member.Role,
		Status:   member.Status,
		JoinedAt: timestamppb.New(member.JoinedAt),
		User:     user,
	}

	orgProto := organizationToProto(&org)

	return connect.NewResponse(&organizationsv1.AcceptInviteResponse{
		Member:       om,
		Organization: orgProto,
	}), nil
}

func (s *Service) DeclineInvite(ctx context.Context, req *connect.Request[organizationsv1.DeclineInviteRequest]) (*connect.Response[organizationsv1.DeclineInviteResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if user.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user email is required to decline invite"))
	}

	// Find the pending invite
	var member database.OrganizationMember
	if err := database.DB.Where("id = ? AND organization_id = ? AND status = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId(), "invited").First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invite not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decline invite: %w", err))
	}

	// Verify the invite is for this user's email (case-insensitive)
	pendingUserID := "pending:" + strings.ToLower(user.Email)
	if !strings.EqualFold(member.UserID, pendingUserID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("this invite is not for your email address"))
	}

	// Delete the invite (declining removes it)
	if err := database.DB.Delete(&member).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decline invite: %w", err))
	}

	return connect.NewResponse(&organizationsv1.DeclineInviteResponse{Success: true}), nil
}

func (s *Service) UpdateMember(ctx context.Context, req *connect.Request[organizationsv1.UpdateMemberRequest]) (*connect.Response[organizationsv1.UpdateMemberResponse], error) {
	actor, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if err := s.authorizeOrgRoles(ctx, req.Msg.GetOrganizationId(), actor, "owner", "admin"); err != nil {
		return nil, err
	}

	isSuper := auth.HasRole(actor, auth.RoleSuperAdmin)

	var m database.OrganizationMember
	if err := database.DB.First(&m, "id = ? AND organization_id = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member not found"))
	}

	requestedRole := strings.TrimSpace(req.Msg.GetRole())
	if requestedRole != "" {
		if strings.EqualFold(requestedRole, "owner") {
			if !isSuper {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("use transfer ownership flow to assign owner"))
			}
			m.Role = "owner"
		} else {
			if strings.EqualFold(m.Role, "owner") {
				if !isSuper {
					return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("the last owner cannot be demoted"))
				}
				var ownerCount int64
				if err := database.DB.Model(&database.OrganizationMember{}).
					Where("organization_id = ? AND role = ?", req.Msg.GetOrganizationId(), "owner").
					Count(&ownerCount).Error; err != nil {
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check owners: %w", err))
				}
				if ownerCount <= 1 {
					return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("organization must retain at least one owner"))
				}
			}
			m.Role = strings.ToLower(requestedRole)
		}
	}

	m.Status = "active"
	if err := database.DB.Save(&m).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update member: %w", err))
	}

	om := &organizationsv1.OrganizationMember{Id: m.ID, Role: m.Role, Status: m.Status, JoinedAt: timestamppb.New(m.JoinedAt), User: &authv1.User{Id: m.UserID}}
	return connect.NewResponse(&organizationsv1.UpdateMemberResponse{Member: om}), nil
}

func (s *Service) RemoveMember(ctx context.Context, req *connect.Request[organizationsv1.RemoveMemberRequest]) (*connect.Response[organizationsv1.RemoveMemberResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if err := s.authorizeOrgRoles(ctx, req.Msg.GetOrganizationId(), user, "owner", "admin"); err != nil {
		return nil, err
	}

	var member database.OrganizationMember
	if err := database.DB.First(&member, "id = ? AND organization_id = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member not found"))
	}

	// Prevent removing yourself (only applies to active members, not invited ones)
	// Invited members have UserID like "pending:email@example.com", which won't match the current user's ID
	if !strings.HasPrefix(member.UserID, "pending:") && member.UserID == user.Id {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("cannot remove yourself from the organization"))
	}

	// Prevent removing the last owner
	if strings.EqualFold(member.Role, "owner") {
		var ownerCount int64
		if err := database.DB.Model(&database.OrganizationMember{}).
			Where("organization_id = ? AND role = ?", req.Msg.GetOrganizationId(), "owner").
			Count(&ownerCount).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check owners: %w", err))
		}
		if ownerCount <= 1 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("organization must retain at least one owner"))
		}
	}

	// Remove the member (works for both active members and invited members)
	if err := database.DB.Delete(&member).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("remove member: %w", err))
	}
	return connect.NewResponse(&organizationsv1.RemoveMemberResponse{Success: true}), nil
}

func (s *Service) TransferOwnership(ctx context.Context, req *connect.Request[organizationsv1.TransferOwnershipRequest]) (*connect.Response[organizationsv1.TransferOwnershipResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	newOwnerMemberID := strings.TrimSpace(req.Msg.GetNewOwnerMemberId())
	if orgID == "" || newOwnerMemberID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id and new_owner_member_id are required"))
	}

	fallbackRole := strings.ToLower(strings.TrimSpace(req.Msg.GetFallbackRole()))
	if fallbackRole == "" {
		fallbackRole = defaultOwnerFallbackRole
	}
	if fallbackRole == "owner" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("fallback role cannot be owner"))
	}

	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := s.authorizeOrgRoles(ctx, orgID, user, "owner"); err != nil {
			return nil, err
		}
	}

	var response *organizationsv1.TransferOwnershipResponse
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var currentOwner database.OrganizationMember
		ownerQuery := tx.Where("organization_id = ? AND role = ?", orgID, "owner")
		if isSuperAdmin {
			ownerQuery = ownerQuery.Where("id <> ?", newOwnerMemberID)
		} else {
			ownerQuery = ownerQuery.Where("user_id = ?", user.Id)
		}

		if err := ownerQuery.First(&currentOwner).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if isSuperAdmin {
					return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no current owner available to transfer from"))
				}
				return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("only current owners can transfer ownership"))
			}
			return fmt.Errorf("lookup current owner: %w", err)
		}
		if currentOwner.ID == newOwnerMemberID {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cannot transfer ownership to yourself"))
		}

		var targetMember database.OrganizationMember
		if err := tx.First(&targetMember, "id = ? AND organization_id = ?", newOwnerMemberID, orgID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return connect.NewError(connect.CodeNotFound, fmt.Errorf("member not found"))
			}
			return fmt.Errorf("lookup target member: %w", err)
		}
		if !strings.EqualFold(targetMember.Status, "active") {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("member must be active before receiving ownership"))
		}
		if strings.EqualFold(targetMember.Role, "owner") {
			return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("member is already an owner"))
		}

		targetMember.Role = "owner"
		targetMember.Status = "active"
		if err := tx.Save(&targetMember).Error; err != nil {
			return fmt.Errorf("promote new owner: %w", err)
		}

		currentOwner.Role = fallbackRole
		if err := tx.Save(&currentOwner).Error; err != nil {
			return fmt.Errorf("update previous owner role: %w", err)
		}

		response = &organizationsv1.TransferOwnershipResponse{
			Success:               true,
			PreviousOwnerMemberId: currentOwner.ID,
			NewOwnerMemberId:      targetMember.ID,
			FallbackRole:          fallbackRole,
		}
		return nil
	}); err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, connectErr
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("transfer ownership: %w", err))
	}

	return connect.NewResponse(response), nil
}

func (s *Service) GetUsage(ctx context.Context, req *connect.Request[organizationsv1.GetUsageRequest]) (*connect.Response[organizationsv1.GetUsageResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := s.authorizeOrgRoles(ctx, orgID, user, "viewer", "member", "admin", "owner"); err != nil {
			return nil, err
		}
	}

	// Determine month (default to current month)
	month := strings.TrimSpace(req.Msg.GetMonth())
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}

	// Calculate estimated monthly usage based on current month progress
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	
	// Calculate elapsed ratio for current month cost prorating
	var elapsedRatio float64
	if month == now.Format("2006-01") {
		elapsed := now.Sub(monthStart)
		monthDuration := monthEnd.Sub(monthStart)
		elapsedRatio = float64(elapsed) / float64(monthDuration)
	} else {
		// Historical month: use full month (1.0) for prorating
		elapsedRatio = 1.0
	}
	
	// Parse requested month for historical queries
	requestedMonthStart := monthStart
	if month != now.Format("2006-01") {
		// Parse historical month
		t, err := time.Parse("2006-01", month)
		if err == nil {
			requestedMonthStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			monthEnd = requestedMonthStart.AddDate(0, 1, 0).Add(-time.Second)
		}
	}
	
	// Calculate usage from deployment_usage_hourly (single source of truth)
	// This works for both current and historical months
	var currentCPUCoreSeconds int64
	var currentMemoryByteSeconds int64
	var currentBandwidthRxBytes int64
	var currentBandwidthTxBytes int64
	var currentStorageBytes int64
	var deploymentsActivePeak int

	if month == now.Format("2006-01") {
		// Current month: calculate live from hourly aggregates (full month) + raw metrics (current incomplete hour)
		
		// Aggregate cutoff: current hour (aggregates exist up to current hour)
		aggregateCutoff := time.Now().UTC().Truncate(time.Hour)
		if aggregateCutoff.Before(monthStart) {
			aggregateCutoff = monthStart
		}

		// Get usage from hourly aggregates for all hours up to (but not including) current hour
		// Use MetricsDB (TimescaleDB) for deployment_usage_hourly
		var hourlyUsage struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
		}
		metricsDB := database.GetMetricsDB()
		if metricsDB == nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("metrics database not available"))
		}
		metricsDB.Table("deployment_usage_hourly duh").
			Select(`
				COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
				COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
				COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("duh.organization_id = ? AND duh.hour >= ? AND duh.hour <= ?", orgID, monthStart, aggregateCutoff).
			Scan(&hourlyUsage)
		
		// Check if aggregates exist for the current hour (aggregateCutoff)
		// If they do, we should exclude raw metrics for that hour to avoid double-counting
		var currentHourAggregateCount int64
		metricsDB.Table("deployment_usage_hourly duh").
			Where("duh.organization_id = ? AND duh.hour = ?", orgID, aggregateCutoff).
			Count(&currentHourAggregateCount)
		
		// Raw metrics start time: if aggregates exist for current hour, start from next hour
		// Otherwise, start from current hour (aggregateCutoff)
		rawMetricsStart := aggregateCutoff
		if currentHourAggregateCount > 0 {
			// Aggregates exist for current hour - only get raw metrics from next hour onwards
			rawMetricsStart = aggregateCutoff.Add(1 * time.Hour)
		}
		
		// Debug: Check if aggregates exist at all for this org
		var aggregateCount int64
		metricsDB.Table("deployment_usage_hourly duh").
			Where("duh.organization_id = ? AND duh.hour >= ? AND duh.hour <= ?", orgID, monthStart, aggregateCutoff).
			Count(&aggregateCount)
		
		logger.Debug("[Organizations] Aggregates for org %s: CPU=%d, Memory=%d bytes, Bandwidth=%d bytes, Hours=%s to %s (inclusive), Count=%d, CurrentHourAggregates=%d", 
			orgID, hourlyUsage.CPUCoreSeconds, hourlyUsage.MemoryByteSeconds, 
			hourlyUsage.BandwidthRxBytes+hourlyUsage.BandwidthTxBytes, monthStart, aggregateCutoff, aggregateCount, currentHourAggregateCount)

		// Get recent usage from raw metrics (current incomplete hour only - not yet aggregated)
		// Raw metrics are only needed for the current incomplete hour since aggregates exist up to current hour
		// Use the SAME logic as deployment-level service: query from aggregateCutoff (current hour start)
		// First get deployment IDs for this organization from main DB, then query metrics from MetricsDB
		var deploymentIDs []string
		database.DB.Table("deployments d").
			Select("d.id").
			Where("d.organization_id = ?", orgID).
			Pluck("id", &deploymentIDs)
		
		// Calculate CPU and Memory from raw metrics per deployment (same approach as deployment-level)
		// Only get metrics from aggregateCutoff onwards (current incomplete hour)
		type metricTimestamp struct {
			CPUUsage    float64
			MemorySum   int64
			Timestamp   time.Time
		}
		var recentCPUFloat float64 // Use float64 to avoid truncation of small values
		var recentMemory int64
		
		if len(deploymentIDs) > 0 {
			// Process each deployment separately to avoid double-counting
			// Only get raw metrics from aggregateCutoff (current hour start) onwards
			for _, deploymentID := range deploymentIDs {
				var deploymentTimestamps []metricTimestamp
				metricsDB.Table("deployment_metrics dm").
					Select(`
						AVG(dm.cpu_usage) as cpu_usage,
						SUM(dm.memory_usage) as memory_sum,
						dm.timestamp as timestamp
					`).
					Where("dm.deployment_id = ? AND dm.timestamp >= ?", deploymentID, rawMetricsStart).
					Group("dm.timestamp").
					Order("dm.timestamp ASC").
					Scan(&deploymentTimestamps)
				
				// Calculate byte-seconds from timestamped metrics (same logic as deployment service)
				metricInterval := int64(5)
				if len(deploymentTimestamps) > 0 {
					// First timestamp: use time from rawMetricsStart to first timestamp, or default interval
					firstTimestamp := deploymentTimestamps[0].Timestamp
					firstInterval := int64(firstTimestamp.Sub(rawMetricsStart).Seconds())
					if firstInterval <= 0 {
						firstInterval = metricInterval
					} else if firstInterval > 3600 {
						firstInterval = metricInterval // Sanity check
					}
					recentCPUFloat += (deploymentTimestamps[0].CPUUsage / 100.0) * float64(firstInterval)
					recentMemory += deploymentTimestamps[0].MemorySum * firstInterval
					
					// Subsequent timestamps: use actual interval between timestamps
					// For each interval from timestamps[i-1] to timestamps[i], use memory[i-1] (the value at the start of the interval)
					for i := 1; i < len(deploymentTimestamps); i++ {
						interval := metricInterval
						intervalSeconds := int64(deploymentTimestamps[i].Timestamp.Sub(deploymentTimestamps[i-1].Timestamp).Seconds())
						if intervalSeconds > 0 && intervalSeconds <= 3600 {
							interval = intervalSeconds
						}
						// Use memory from the PREVIOUS timestamp for this interval
						recentCPUFloat += (deploymentTimestamps[i-1].CPUUsage / 100.0) * float64(interval)
						recentMemory += deploymentTimestamps[i-1].MemorySum * interval
					}
				}
			}
		}
		recentCPU := int64(recentCPUFloat) // Convert to int64 at the end

		// Get bandwidth from raw metrics (current incomplete hour only)
		// Use MetricsDB (TimescaleDB) for deployment_metrics
		var recentBandwidth struct {
			BandwidthRxBytes int64
			BandwidthTxBytes int64
		}
		if len(deploymentIDs) > 0 {
			metricsDB.Table("deployment_metrics dm").
				Select(`
					COALESCE(SUM(dm.network_rx_bytes), 0) as bandwidth_rx_bytes,
					COALESCE(SUM(dm.network_tx_bytes), 0) as bandwidth_tx_bytes
				`).
				Where("dm.deployment_id IN ? AND dm.timestamp >= ?", deploymentIDs, rawMetricsStart).
				Scan(&recentBandwidth)
		}

		// Combine: hourly aggregates (all hours up to current hour) + raw metrics (current incomplete hour) = live current month usage
		currentCPUCoreSeconds = hourlyUsage.CPUCoreSeconds + recentCPU
		currentMemoryByteSeconds = hourlyUsage.MemoryByteSeconds + recentMemory
		currentBandwidthRxBytes = hourlyUsage.BandwidthRxBytes + recentBandwidth.BandwidthRxBytes
		currentBandwidthTxBytes = hourlyUsage.BandwidthTxBytes + recentBandwidth.BandwidthTxBytes
		
		logger.Debug("[Organizations] Combined for org %s: CPU=%d (agg=%d + raw=%d), Memory=%d bytes (agg=%d + raw=%d), Bandwidth=%d bytes (agg=%d + raw=%d)", 
			orgID, currentCPUCoreSeconds, hourlyUsage.CPUCoreSeconds, recentCPU,
			currentMemoryByteSeconds, hourlyUsage.MemoryByteSeconds, recentMemory,
			currentBandwidthRxBytes+currentBandwidthTxBytes, hourlyUsage.BandwidthRxBytes+hourlyUsage.BandwidthTxBytes, recentBandwidth.BandwidthRxBytes+recentBandwidth.BandwidthTxBytes)
		var storageSum struct {
			StorageBytes int64
		}
		database.DB.Table("deployments d").
			Select("COALESCE(SUM(d.storage_bytes), 0) as storage_bytes").
			Where("d.organization_id = ?", orgID).
			Scan(&storageSum)
		currentStorageBytes = storageSum.StorageBytes
		
		// Get peak deployments count for current month
		var peakCount int
		database.DB.Table("deployment_locations dl").
			Select("COUNT(DISTINCT dl.deployment_id)").
			Joins("JOIN deployments d ON d.id = dl.deployment_id").
			Where("d.organization_id = ? AND dl.status = ? AND (dl.created_at >= ? OR dl.updated_at >= ?)", orgID, "running", monthStart, monthStart).
			Scan(&peakCount)
		deploymentsActivePeak = peakCount
	} else {
		// Historical month: calculate from deployment_usage_hourly
		// Get usage from hourly aggregates for the entire requested month
		// Use MetricsDB (TimescaleDB) for deployment_usage_hourly
		var hourlyUsage struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
		}
		metricsDB := database.GetMetricsDB()
		if metricsDB == nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("metrics database not available"))
		}
		metricsDB.Table("deployment_usage_hourly duh").
			Select(`
				COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
				COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
				COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("duh.organization_id = ? AND duh.hour >= ? AND duh.hour <= ?", orgID, requestedMonthStart, monthEnd).
			Scan(&hourlyUsage)
		
		currentCPUCoreSeconds = hourlyUsage.CPUCoreSeconds
		currentMemoryByteSeconds = hourlyUsage.MemoryByteSeconds
		currentBandwidthRxBytes = hourlyUsage.BandwidthRxBytes
		currentBandwidthTxBytes = hourlyUsage.BandwidthTxBytes
		
		// Storage: get snapshot from deployments table for the month (use current, as historical storage is not tracked)
		var storageSum struct {
			StorageBytes int64
		}
		database.DB.Table("deployments d").
			Select("COALESCE(SUM(d.storage_bytes), 0) as storage_bytes").
			Where("d.organization_id = ?", orgID).
			Scan(&storageSum)
		currentStorageBytes = storageSum.StorageBytes
		
		// Peak deployments: not tracked historically, use 0 or current
		deploymentsActivePeak = 0
	}
	
	var estimatedMonthly *organizationsv1.UsageMetrics
	if month == now.Format("2006-01") {
		// Current month: project based on elapsed time using live calculated values
		// Only project if we have sufficient data (at least 7 days) to avoid massive inflation early in the month
		// Early in the month, elapsedRatio is very small (e.g., 0.033 for 1 day), causing 30x inflation
		// This is inaccurate if deployments only ran briefly
		daysElapsed := float64(now.Sub(monthStart).Hours()) / 24.0
		minDaysForProjection := 7.0 // Only project if we have at least 7 days of data
		
		if elapsedRatio > 0 && elapsedRatio < 1.0 && daysElapsed >= minDaysForProjection {
			// We have sufficient data - project to full month
			estimatedMonthly = &organizationsv1.UsageMetrics{
				CpuCoreSeconds:      int64(float64(currentCPUCoreSeconds) / elapsedRatio),
				MemoryByteSeconds:   int64(float64(currentMemoryByteSeconds) / elapsedRatio),
				BandwidthRxBytes:    currentBandwidthRxBytes, // Bandwidth is cumulative, use current value for estimate
				BandwidthTxBytes:    currentBandwidthTxBytes,
				StorageBytes:        currentStorageBytes, // Storage is snapshot, use current value for estimate
				DeploymentsActivePeak: int32(deploymentsActivePeak),
			}
			logger.Debug("[Organizations] Projected for org %s: CPU=%d, Memory=%d bytes (from current=%d, ratio=%.3f, days=%.1f)", 
				orgID, estimatedMonthly.CpuCoreSeconds, estimatedMonthly.MemoryByteSeconds, 
				currentMemoryByteSeconds, elapsedRatio, daysElapsed)
		} else {
			// Not enough data for accurate projection - use current usage as estimate
			// This avoids massive inflation early in the month when deployments may have only run briefly
			estimatedMonthly = &organizationsv1.UsageMetrics{
				CpuCoreSeconds:      currentCPUCoreSeconds,
				MemoryByteSeconds:   currentMemoryByteSeconds,
				BandwidthRxBytes:    currentBandwidthRxBytes,
				BandwidthTxBytes:    currentBandwidthTxBytes,
				StorageBytes:        currentStorageBytes,
				DeploymentsActivePeak: int32(deploymentsActivePeak),
			}
			if daysElapsed < minDaysForProjection {
				logger.Debug("[Organizations] Not projecting for org %s: only %.1f days elapsed (min=%d), using current usage as estimate", 
					orgID, daysElapsed, int(minDaysForProjection))
			}
		}
	} else {
		// Historical month: estimated equals current (from aggregated data)
		estimatedMonthly = &organizationsv1.UsageMetrics{
			CpuCoreSeconds:      currentCPUCoreSeconds,
			MemoryByteSeconds:   currentMemoryByteSeconds,
			BandwidthRxBytes:    currentBandwidthRxBytes,
			BandwidthTxBytes:    currentBandwidthTxBytes,
			StorageBytes:        currentStorageBytes,
			DeploymentsActivePeak: int32(deploymentsActivePeak),
		}
	}

	// Calculate estimated cost using centralized pricing model
	pricingModel := pricing.GetPricing()
	bandwidthBytes := estimatedMonthly.BandwidthRxBytes + estimatedMonthly.BandwidthTxBytes
	
	// Calculate per-resource costs for estimated monthly
	estCPUCost := pricingModel.CalculateCPUCost(estimatedMonthly.CpuCoreSeconds)
	estMemoryCost := pricingModel.CalculateMemoryCost(estimatedMonthly.MemoryByteSeconds)
	estBandwidthCost := pricingModel.CalculateBandwidthCost(bandwidthBytes)
	estStorageCost := pricingModel.CalculateStorageCost(estimatedMonthly.StorageBytes)
	estimatedCostCents := estCPUCost + estMemoryCost + estBandwidthCost + estStorageCost

	// Calculate current cost using centralized pricing model with live calculated values
	// Note: Storage is billed monthly - for current cost, we need to prorate it
	// CPU/Memory are already time-based (core-seconds, byte-seconds) so no prorating needed
	// Bandwidth is one-time cost per byte transferred, no prorating needed
	// Storage is monthly cost per byte, so must prorate based on elapsed time
	currBandwidthBytes := currentBandwidthRxBytes + currentBandwidthTxBytes
	cpuCost := pricingModel.CalculateCPUCost(currentCPUCoreSeconds)
	memoryCost := pricingModel.CalculateMemoryCost(currentMemoryByteSeconds)
	bandwidthCost := pricingModel.CalculateBandwidthCost(currBandwidthBytes)
	
	// Storage cost is monthly rate, prorate for current cost calculation
	var currentCostCents int64
	var currentStorageCost int64
	if month == now.Format("2006-01") && elapsedRatio > 0 {
		storageCostFullMonth := pricingModel.CalculateStorageCost(currentStorageBytes)
		currentStorageCost = int64(float64(storageCostFullMonth) * elapsedRatio)
		currentCostCents = cpuCost + memoryCost + bandwidthCost + currentStorageCost
	} else {
		// Historical month: storage is already for full month
		currentStorageCost = pricingModel.CalculateStorageCost(currentStorageBytes)
		currentCostCents = cpuCost + memoryCost + bandwidthCost + currentStorageCost
	}

	// Set per-resource cost breakdown for estimated monthly
	cpuCostPtr := int64(estCPUCost)
	memoryCostPtr := int64(estMemoryCost)
	bandwidthCostPtr := int64(estBandwidthCost)
	storageCostPtr := int64(estStorageCost)
	estimatedMonthly.EstimatedCostCents = estimatedCostCents
	estimatedMonthly.CpuCostCents = &cpuCostPtr
	estimatedMonthly.MemoryCostCents = &memoryCostPtr
	estimatedMonthly.BandwidthCostCents = &bandwidthCostPtr
	estimatedMonthly.StorageCostCents = &storageCostPtr
	
	logger.Debug("[Organizations] Cost breakdown for org %s: CPU=%d cents (%.2f), Memory=%d cents (%.2f), Bandwidth=%d cents (%.2f), Storage=%d cents (%.2f), Total=%d cents (%.2f)",
		orgID, cpuCostPtr, float64(cpuCostPtr)/100, memoryCostPtr, float64(memoryCostPtr)/100,
		bandwidthCostPtr, float64(bandwidthCostPtr)/100, storageCostPtr, float64(storageCostPtr)/100,
		estimatedCostCents, float64(estimatedCostCents)/100)

	// Set per-resource cost breakdown for current usage
	currCPUCostPtr := int64(cpuCost)
	currMemoryCostPtr := int64(memoryCost)
	currBandwidthCostPtr := int64(bandwidthCost)
	currStorageCostPtr := currentStorageCost

	currentMetrics := &organizationsv1.UsageMetrics{
		CpuCoreSeconds:      currentCPUCoreSeconds,
		MemoryByteSeconds:   currentMemoryByteSeconds,
		BandwidthRxBytes:    currentBandwidthRxBytes,
		BandwidthTxBytes:    currentBandwidthTxBytes,
		StorageBytes:        currentStorageBytes,
		DeploymentsActivePeak: int32(deploymentsActivePeak),
		EstimatedCostCents: currentCostCents, // Current usage cost (calculated server-side with live data)
		CpuCostCents:        &currCPUCostPtr,
		MemoryCostCents:     &currMemoryCostPtr,
		BandwidthCostCents:  &currBandwidthCostPtr,
		StorageCostCents:    &currStorageCostPtr,
	}

	// Get quota information
	var quota *organizationsv1.UsageQuota
	var orgQuota database.OrgQuota
	if err := database.DB.First(&orgQuota, "organization_id = ?", orgID).Error; err == nil {
		// Get plan details
		var plan database.OrganizationPlan
		if err := database.DB.First(&plan, "id = ?", orgQuota.PlanID).Error; err == nil {
			cpuLimit := plan.CPUCores
			if orgQuota.CPUCoresOverride != nil && *orgQuota.CPUCoresOverride > 0 {
				cpuLimit = *orgQuota.CPUCoresOverride
			}
			
			memoryLimit := plan.MemoryBytes
			if orgQuota.MemoryBytesOverride != nil && *orgQuota.MemoryBytesOverride > 0 {
				memoryLimit = *orgQuota.MemoryBytesOverride
			}
			
			bandwidthLimit := plan.BandwidthBytesMonth
			if orgQuota.BandwidthBytesMonthOverride != nil && *orgQuota.BandwidthBytesMonthOverride > 0 {
				bandwidthLimit = *orgQuota.BandwidthBytesMonthOverride
			}
			
			storageLimit := plan.StorageBytes
			if orgQuota.StorageBytesOverride != nil && *orgQuota.StorageBytesOverride > 0 {
				storageLimit = *orgQuota.StorageBytesOverride
			}
			
			deploymentsMax := plan.DeploymentsMax
			if orgQuota.DeploymentsMaxOverride != nil && *orgQuota.DeploymentsMaxOverride > 0 {
				deploymentsMax = *orgQuota.DeploymentsMaxOverride
			}

			// Convert to monthly limits (CPU and Memory are per-second, so multiply by seconds in month)
			secondsInMonth := int64(monthEnd.Sub(monthStart).Seconds())
			quota = &organizationsv1.UsageQuota{
				CpuCoreSecondsMonthly:   int64(cpuLimit) * secondsInMonth,
				MemoryByteSecondsMonthly: memoryLimit * secondsInMonth,
				BandwidthBytesMonthly:    bandwidthLimit,
				StorageBytes:             storageLimit,
				DeploymentsMax:           int32(deploymentsMax),
			}
		}
	}
	
	if quota == nil {
		// Default quota (0 = unlimited)
		quota = &organizationsv1.UsageQuota{
			CpuCoreSecondsMonthly:   0,
			MemoryByteSecondsMonthly: 0,
			BandwidthBytesMonthly:    0,
			StorageBytes:             0,
			DeploymentsMax:           0,
		}
	}

	response := &organizationsv1.GetUsageResponse{
		OrganizationId:    orgID,
		Month:             month,
		Current:           currentMetrics,
		EstimatedMonthly:  estimatedMonthly,
		Quota:             quota,
	}

	return connect.NewResponse(response), nil
}

func ensurePersonalOrg(userID string) {
	// Check membership
	var count int64
	database.DB.Model(&database.OrganizationMember{}).Where("user_id = ?", userID).Count(&count)
	if count > 0 {
		return
	}
	// Create personal org
	now := time.Now()
	org := &database.Organization{ID: generateID("org"), Name: "Personal", Slug: "personal-" + userID, Plan: "personal", Status: "active", CreatedAt: now}
	if err := database.DB.Create(org).Error; err != nil {
		return
	}
	m := &database.OrganizationMember{ID: generateID("mem"), OrganizationID: org.ID, UserID: userID, Role: "owner", Status: "active", JoinedAt: now}
	_ = database.DB.Create(m).Error
	// Ensure organization has a plan assigned (defaults to Starter plan)
	_ = EnsurePlanAssigned(org.ID)
}

func generateID(prefix string) string { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()) }

// organizationToProto converts a database Organization to a proto Organization, including plan info
func organizationToProto(org *database.Organization) *organizationsv1.Organization {
	po := &organizationsv1.Organization{
		Id:            org.ID,
		Name:          org.Name,
		Slug:          org.Slug,
		Plan:          strings.ToLower(org.Plan),
		Status:        org.Status,
		Credits:       org.Credits,
		TotalPaidCents: org.TotalPaidCents,
		CreatedAt:     timestamppb.New(org.CreatedAt),
	}
	if org.Domain != nil {
		po.Domain = org.Domain
	}

	// Load plan info if organization has a plan assigned
	var quota database.OrgQuota
	if err := database.DB.First(&quota, "organization_id = ?", org.ID).Error; err != nil {
		// No quota found - this is expected if plan hasn't been assigned yet
		// EnsurePlanAssigned should have been called before this, but if not, we'll just skip plan info
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[organizationToProto] No quota found for org %s (this is expected if EnsurePlanAssigned wasn't called or failed)", org.ID)
		} else {
			log.Printf("[organizationToProto] Error loading quota for org %s: %v", org.ID, err)
		}
	} else {
		log.Printf("[organizationToProto] Found quota for org %s: PlanID=%s", org.ID, quota.PlanID)
		if quota.PlanID != "" {
			var plan database.OrganizationPlan
			if err := database.DB.First(&plan, "id = ?", quota.PlanID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("[organizationToProto] Plan %s not found for org %s (quota references non-existent plan)", quota.PlanID, org.ID)
				} else {
					log.Printf("[organizationToProto] Error loading plan %s for org %s: %v", quota.PlanID, org.ID, err)
				}
			} else {
				log.Printf("[organizationToProto] Successfully loaded plan %s (%s) for org %s", plan.ID, plan.Name, org.ID)
				po.PlanInfo = &organizationsv1.PlanInfo{
					PlanId:                  plan.ID,
					PlanName:                plan.Name,
					Description:             plan.Description,
					CpuCores:                int32(plan.CPUCores),
					MemoryBytes:             plan.MemoryBytes,
					DeploymentsMax:          int32(plan.DeploymentsMax),
					BandwidthBytesMonth:     plan.BandwidthBytesMonth,
					StorageBytes:            plan.StorageBytes,
					MinimumPaymentCents:     plan.MinimumPaymentCents,
					MonthlyFreeCreditsCents: plan.MonthlyFreeCreditsCents,
				}
			}
		} else {
			log.Printf("[organizationToProto] Quota exists for org %s but PlanID is empty", org.ID)
		}
	}

	return po
}

func normalizeSlug(input string) string {
	lowered := strings.ToLower(strings.TrimSpace(input))
	if lowered == "" {
		return "organization"
	}

	cleaned := strings.Builder{}
	for _, r := range lowered {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cleaned.WriteRune(r)
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if cleaned.Len() > 0 && cleaned.String()[cleaned.Len()-1] != '-' {
				cleaned.WriteRune('-')
			}
		}
	}

	slug := cleaned.String()
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "organization"
	}

	return slug
}

func (s *Service) authorizeOrgRoles(ctx context.Context, orgID string, user *authv1.User, allowedRoles ...string) error {
	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil
	}

	var member database.OrganizationMember
	if err := database.DB.First(&member, "organization_id = ? AND user_id = ?", orgID, user.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this organization"))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("membership lookup: %w", err))
	}

	if !strings.EqualFold(member.Status, "active") {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("inactive members cannot perform this action"))
	}

	if len(allowedRoles) == 0 {
		return nil
	}

	role := strings.ToLower(member.Role)
	for _, allowed := range allowedRoles {
		if role == strings.ToLower(allowed) {
			return nil
		}
	}

	return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient role to perform this action"))
}

// plan limits are enforced via quotas; not needed here

func deriveNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 0 || parts[0] == "" {
		return "New Team Member"
	}

	tokens := strings.FieldsFunc(parts[0], func(r rune) bool { return r == '.' || r == '_' || r == '-' })
	for i, token := range tokens {
		if token == "" {
			continue
		}
		tokens[i] = strings.ToUpper(token[:1]) + token[1:]
	}

	name := strings.Join(tokens, " ")
	if name == "" {
		return "New Team Member"
	}

	return name
}

// dispatchInviteEmail sends an invite email and returns an error if sending fails.
// If email sending succeeds, it updates the member's LastInviteSentAt timestamp.
func (s *Service) dispatchInviteEmail(ctx context.Context, org *database.Organization, member *database.OrganizationMember, inviter *authv1.User, inviteeEmail string) error {
	if s.mailer == nil || !s.mailer.Enabled() {
		return nil // Not an error if email is disabled
	}

	consoleURL := s.consoleURL
	if consoleURL == "" {
		consoleURL = defaultConsoleURL
	}

	inviterName := strings.TrimSpace(inviter.GetName())
	if inviterName == "" {
		inviterName = strings.TrimSpace(inviter.GetEmail())
	}

	roleLabel := capitalize(member.Role)
	greetingName := deriveNameFromEmail(inviteeEmail)

	subject := fmt.Sprintf("%s invited you to %s on Obiente Cloud", inviterName, org.Name)
	template := email.TemplateData{
		Subject:     subject,
		PreviewText: fmt.Sprintf("Join %s on Obiente Cloud.", org.Name),
		Greeting:    fmt.Sprintf("Hi %s,", greetingName),
		Heading:     fmt.Sprintf("You're invited to %s", org.Name),
		IntroLines: []string{
			fmt.Sprintf("%s has invited you to collaborate with the %s organization on Obiente Cloud.", inviterName, org.Name),
			"Accept the invite to access your team's projects, environments, and billing details in one place.",
		},
		Highlights: []email.Highlight{
			{Label: "Organization", Value: org.Name},
			{Label: "Role", Value: roleLabel},
		},
		Sections: []email.Section{
			{
				Title: "Next steps",
				Lines: []string{
					fmt.Sprintf("Sign in at %s using %s.", consoleURL, inviteeEmail),
					"The invitation will be waiting on your dashboard - just confirm to activate access.",
				},
			},
		},
		CTA: &email.CTA{
			Label:       "Accept invitation",
			URL:         consoleURL,
			Description: "Sign in with your invitation email to finish onboarding.",
		},
		SignatureLines: []string{
			"See you in the cloud,",
			"The Obiente Cloud Team",
		},
		SupportEmail: s.supportEmail,
		BrandURL:     consoleURL,
		BaseURL:      consoleURL,
		Category:     email.CategoryInvite,
	}

	message := &email.Message{
		To:       []string{inviteeEmail},
		Subject:  subject,
		Template: &template,
		Category: email.CategoryInvite,
		Metadata: map[string]string{
			"organization-id":   org.ID,
			"organization-role": member.Role,
			"inviter-id":        inviter.GetId(),
		},
	}

	if err := s.mailer.Send(ctx, message); err != nil {
		log.Printf("[Organizations] failed to send invite email for member %s: %v", member.ID, err)
		return err
	}

	// Only update LastInviteSentAt if email was successfully sent
	now := time.Now()
	member.LastInviteSentAt = &now
	if err := database.DB.Model(member).Update("last_invite_sent_at", now).Error; err != nil {
		log.Printf("[Organizations] failed to update LastInviteSentAt for member %s: %v", member.ID, err)
		// Don't fail the whole operation if timestamp update fails
	}

	return nil
}

func capitalize(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	lowered := strings.ToLower(trimmed)
	return strings.ToUpper(lowered[:1]) + lowered[1:]
}

func buildUserProfile(ctx context.Context, resolver *userProfileResolver, member database.OrganizationMember) *authv1.User {
	userProto := &authv1.User{Id: member.UserID}

	if strings.HasPrefix(member.UserID, "pending:") {
		email := strings.TrimPrefix(member.UserID, "pending:")
		userProto.Email = email
		userProto.Name = deriveNameFromEmail(email)
		userProto.PreferredUsername = email
		return userProto
	}

	if resolver != nil {
		if profile, err := resolver.Resolve(ctx, member.UserID); err == nil && profile != nil {
			if profile.Id == "" {
				profile.Id = member.UserID
			}
			return profile
		} else if err != nil {
			log.Printf("[Organizations] failed to resolve profile for %s: %v", member.UserID, err)
		}
	}

	// Fallback: if we couldn't resolve, at least set a name from the ID
	if userProto.Name == "" {
		userProto.Name = member.UserID
	}

	return userProto
}

func (s *Service) AddCredits(ctx context.Context, req *connect.Request[organizationsv1.AddCreditsRequest]) (*connect.Response[organizationsv1.AddCreditsResponse], error) {
	// SECURITY: Users should not be able to add credits without payment
	// This endpoint is deprecated - use AdminAddCredits or payment processing instead
	return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("users cannot add credits directly. Credits must be added through payment processing or by administrators"))
}

func (s *Service) AdminAddCredits(ctx context.Context, req *connect.Request[organizationsv1.AdminAddCreditsRequest]) (*connect.Response[organizationsv1.AdminAddCreditsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	// Only superadmins can use this
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	amountCents := req.Msg.GetAmountCents()
	if amountCents <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount_cents must be positive"))
	}

	// Update credits in a transaction and record it
	var org database.Organization
	var note *string
	if req.Msg.GetNote() != "" {
		n := req.Msg.GetNote()
		note = &n
	}
	userID := user.Id

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
			return err
		}
		org.Credits += amountCents
		if org.Credits < 0 {
			org.Credits = 0 // Prevent negative balances
		}
		if err := tx.Save(&org).Error; err != nil {
			return err
		}
		// Record transaction in credit log
		transaction := &database.CreditTransaction{
			ID:             generateID("ct"),
			OrganizationID: orgID,
			AmountCents:    amountCents,
			BalanceAfter:   org.Credits,
			Type:           "admin_add",
			Source:         "admin",
			Note:           note,
			CreatedBy:      &userID,
			CreatedAt:      time.Now(),
		}
		return tx.Create(transaction).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("admin add credits: %w", err))
	}

	// Ensure organization has a plan assigned (defaults to Starter plan)
	_ = EnsurePlanAssigned(orgID)

	return connect.NewResponse(&organizationsv1.AdminAddCreditsResponse{
		Organization:      organizationToProto(&org),
		NewBalanceCents:   org.Credits,
		AmountAddedCents: amountCents,
	}), nil
}

func (s *Service) AdminRemoveCredits(ctx context.Context, req *connect.Request[organizationsv1.AdminRemoveCreditsRequest]) (*connect.Response[organizationsv1.AdminRemoveCreditsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	// Only superadmins can use this
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	amountCents := req.Msg.GetAmountCents()
	if amountCents <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount_cents must be positive"))
	}

	// Update credits in a transaction and record it
	var org database.Organization
	var note *string
	if req.Msg.GetNote() != "" {
		n := req.Msg.GetNote()
		note = &n
	}
	userID := user.Id
	var oldBalance int64
	var actualRemoved int64

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
			return err
		}
		oldBalance = org.Credits
		org.Credits -= amountCents
		if org.Credits < 0 {
			org.Credits = 0 // Prevent negative balances
		}
		actualRemoved = oldBalance - org.Credits
		if err := tx.Save(&org).Error; err != nil {
			return err
		}
		// Record transaction in credit log (negative amount for removal)
		transaction := &database.CreditTransaction{
			ID:             generateID("ct"),
			OrganizationID: orgID,
			AmountCents:    -actualRemoved, // Negative for removal
			BalanceAfter:   org.Credits,
			Type:           "admin_remove",
			Source:         "admin",
			Note:           note,
			CreatedBy:      &userID,
			CreatedAt:      time.Now(),
		}
		return tx.Create(transaction).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("admin remove credits: %w", err))
	}

	return connect.NewResponse(&organizationsv1.AdminRemoveCreditsResponse{
		Organization:        organizationToProto(&org),
		NewBalanceCents:     org.Credits,
		AmountRemovedCents: actualRemoved,
	}), nil
}

func (s *Service) GetCreditLog(ctx context.Context, req *connect.Request[organizationsv1.GetCreditLogRequest]) (*connect.Response[organizationsv1.GetCreditLogResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Check authorization - user must be a member or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("access denied to organization"))
		}
	}

	// Pagination
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Query transactions
	var transactions []database.CreditTransaction
	var total int64

	if err := database.DB.Model(&database.CreditTransaction{}).
		Where("organization_id = ?", orgID).
		Count(&total).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get credit log: %w", err))
	}

	if err := database.DB.Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get credit log: %w", err))
		}

	// Convert to proto
	protoTransactions := make([]*organizationsv1.CreditTransaction, 0, len(transactions))
	for _, t := range transactions {
		pt := &organizationsv1.CreditTransaction{
			Id:           t.ID,
			OrganizationId: t.OrganizationID,
			AmountCents:  t.AmountCents,
			BalanceAfter: t.BalanceAfter,
			Type:         t.Type,
			Source:       t.Source,
			CreatedAt:    timestamppb.New(t.CreatedAt),
	}
		if t.Note != nil {
			pt.Note = t.Note
		}
		if t.CreatedBy != nil {
			pt.CreatedBy = t.CreatedBy
		}
		protoTransactions = append(protoTransactions, pt)
	}

	totalPages := (int(total) + perPage - 1) / perPage

	return connect.NewResponse(&organizationsv1.GetCreditLogResponse{
		Transactions: protoTransactions,
		Pagination: &organizationsv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(total),
			TotalPages: int32(totalPages),
		},
	}), nil
}
