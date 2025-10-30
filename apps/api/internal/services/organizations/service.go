package organizations

import (
	"context"
	"fmt"
	"strings"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	organizationsv1connect.UnimplementedOrganizationServiceHandler
}

func NewService() organizationsv1connect.OrganizationServiceHandler { return &Service{} }

func (s *Service) ListOrganizations(ctx context.Context, _ *connect.Request[organizationsv1.ListOrganizationsRequest]) (*connect.Response[organizationsv1.ListOrganizationsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	// Ensure personal org exists
	ensurePersonalOrg(user.Id)
	// Fetch orgs for user
	type row struct {
		Id, Name, Slug, Plan, Status string
		Domain                       *string
		CreatedAt                    time.Time
	}
	var rows []row
	database.DB.Raw(`
        SELECT o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at
        FROM organizations o
        JOIN organization_members m ON m.organization_id = o.id
        WHERE m.user_id = ?
    `, user.Id).Scan(&rows)
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

func (s *Service) UpdateOrganization(_ context.Context, req *connect.Request[organizationsv1.UpdateOrganizationRequest]) (*connect.Response[organizationsv1.UpdateOrganizationResponse], error) {
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

func (s *Service) ListMembers(_ context.Context, req *connect.Request[organizationsv1.ListMembersRequest]) (*connect.Response[organizationsv1.ListMembersResponse], error) {
	var rows []database.OrganizationMember
	if err := database.DB.Where("organization_id = ?", req.Msg.GetOrganizationId()).Find(&rows).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list members: %w", err))
	}
	list := make([]*organizationsv1.OrganizationMember, 0, len(rows))
	for _, m := range rows {
		om := &organizationsv1.OrganizationMember{Id: m.ID, Role: m.Role, Status: m.Status, JoinedAt: timestamppb.New(m.JoinedAt), User: &authv1.User{Id: m.UserID}}
		list = append(list, om)
	}
	res := connect.NewResponse(&organizationsv1.ListMembersResponse{Members: list, Pagination: &organizationsv1.Pagination{Page: 1, PerPage: int32(len(list)), Total: int32(len(list)), TotalPages: 1}})
	return res, nil
}

func (s *Service) InviteMember(_ context.Context, req *connect.Request[organizationsv1.InviteMemberRequest]) (*connect.Response[organizationsv1.InviteMemberResponse], error) {
	email := strings.TrimSpace(req.Msg.GetEmail())
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("member email is required"))
	}
	role := req.Msg.GetRole()
	if role == "" {
		role = "member"
	}
	m := &database.OrganizationMember{ID: generateID("mem"), OrganizationID: req.Msg.GetOrganizationId(), UserID: "pending:" + email, Role: strings.ToLower(role), Status: "invited", JoinedAt: time.Now()}
	if err := database.DB.Create(m).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invite member: %w", err))
	}
	om := &organizationsv1.OrganizationMember{Id: m.ID, Role: m.Role, Status: m.Status, JoinedAt: timestamppb.New(m.JoinedAt), User: &authv1.User{Id: m.UserID, Email: email, Name: deriveNameFromEmail(email)}}
	return connect.NewResponse(&organizationsv1.InviteMemberResponse{Member: om}), nil
}

func (s *Service) UpdateMember(_ context.Context, req *connect.Request[organizationsv1.UpdateMemberRequest]) (*connect.Response[organizationsv1.UpdateMemberResponse], error) {
	var m database.OrganizationMember
	if err := database.DB.First(&m, "id = ? AND organization_id = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member not found"))
	}
	if role := req.Msg.GetRole(); role != "" {
		m.Role = strings.ToLower(role)
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

	var member database.OrganizationMember
	if err := database.DB.First(&member, "id = ? AND organization_id = ?", req.Msg.GetMemberId(), req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member not found"))
	}

	if member.UserID == user.Id {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("cannot remove yourself from the organization"))
	}

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

	if err := database.DB.Delete(&member).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("remove member: %w", err))
	}
	return connect.NewResponse(&organizationsv1.RemoveMemberResponse{Success: true}), nil
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

// legacy helpers removed after DB-backed refactor

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
