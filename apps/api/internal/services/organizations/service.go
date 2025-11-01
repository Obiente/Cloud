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

func (s *Service) ListOrganizations(ctx context.Context, _ *connect.Request[organizationsv1.ListOrganizationsRequest]) (*connect.Response[organizationsv1.ListOrganizationsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	type row struct {
		Id, Name, Slug, Plan, Status string
		Domain                       *string
		CreatedAt                    time.Time
	}

	var rows []row
	if auth.HasRole(user, auth.RoleSuperAdmin) {
		if err := database.DB.Raw(`
			SELECT o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at
			FROM organizations o
			ORDER BY o.created_at DESC
		`).Scan(&rows).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list organizations: %w", err))
		}
	} else {
		ensurePersonalOrg(user.Id)
		if err := database.DB.Raw(`
			SELECT o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at
			FROM organizations o
			JOIN organization_members m ON m.organization_id = o.id
			WHERE m.user_id = ?
			ORDER BY o.created_at DESC
		`, user.Id).Scan(&rows).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list organizations: %w", err))
		}
	}

	orgs := make([]*organizationsv1.Organization, 0, len(rows))
	for _, r := range rows {
		po := &organizationsv1.Organization{
			Id: r.Id, Name: r.Name, Slug: r.Slug, Plan: strings.ToLower(r.Plan), Status: r.Status,
			CreatedAt: timestamppb.New(r.CreatedAt),
		}
		if r.Domain != nil {
			po.Domain = r.Domain
		}
		orgs = append(orgs, po)
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
	org := &database.Organization{ID: generateID("org"), Name: name, Slug: slug, Plan: plan, Status: "active", CreatedAt: now}
	if err := database.DB.Create(org).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create org: %w", err))
	}
	// add creator as owner
	m := &database.OrganizationMember{ID: generateID("mem"), OrganizationID: org.ID, UserID: user.Id, Role: "owner", Status: "active", JoinedAt: now}
	_ = database.DB.Create(m).Error
	po := &organizationsv1.Organization{Id: org.ID, Name: org.Name, Slug: org.Slug, Plan: strings.ToLower(org.Plan), Status: org.Status, CreatedAt: timestamppb.New(org.CreatedAt)}
	return connect.NewResponse(&organizationsv1.CreateOrganizationResponse{Organization: po}), nil
}

func (s *Service) GetOrganization(_ context.Context, req *connect.Request[organizationsv1.GetOrganizationRequest]) (*connect.Response[organizationsv1.GetOrganizationResponse], error) {
	var r database.Organization
	if err := database.DB.First(&r, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}
	po := &organizationsv1.Organization{Id: r.ID, Name: r.Name, Slug: r.Slug, Plan: strings.ToLower(r.Plan), Status: r.Status, CreatedAt: timestamppb.New(r.CreatedAt)}
	if r.Domain != nil {
		po.Domain = r.Domain
	}
	return connect.NewResponse(&organizationsv1.GetOrganizationResponse{Organization: po}), nil
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
	po := &organizationsv1.Organization{Id: org.ID, Name: org.Name, Slug: org.Slug, Plan: strings.ToLower(org.Plan), Status: org.Status, CreatedAt: timestamppb.New(org.CreatedAt)}
	if org.Domain != nil {
		po.Domain = org.Domain
	}
	return connect.NewResponse(&organizationsv1.UpdateOrganizationResponse{Organization: po}), nil
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

	m := &database.OrganizationMember{ID: generateID("mem"), OrganizationID: org.ID, UserID: "pending:" + emailAddr, Role: strings.ToLower(role), Status: "invited", JoinedAt: time.Now()}
	if err := database.DB.Create(m).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invite member: %w", err))
	}

	s.dispatchInviteEmail(ctx, &org, m, inviter, emailAddr)

	om := &organizationsv1.OrganizationMember{Id: m.ID, Role: m.Role, Status: m.Status, JoinedAt: timestamppb.New(m.JoinedAt), User: &authv1.User{Id: m.UserID, Email: emailAddr, Name: deriveNameFromEmail(emailAddr)}}
	return connect.NewResponse(&organizationsv1.InviteMemberResponse{Member: om}), nil
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
}

func generateID(prefix string) string { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()) }

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

func (s *Service) dispatchInviteEmail(ctx context.Context, org *database.Organization, member *database.OrganizationMember, inviter *authv1.User, inviteeEmail string) {
	if s.mailer == nil || !s.mailer.Enabled() {
		return
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
	}
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

	return userProto
}
