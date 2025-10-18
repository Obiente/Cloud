package organizations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	organizationsv1connect.UnimplementedOrganizationServiceHandler

	mu             sync.RWMutex
	organizations  map[string]*organizationsv1.Organization
	members        map[string]map[string]*organizationsv1.OrganizationMember
	organizationID int
	memberID       int
}

func NewService() organizationsv1connect.OrganizationServiceHandler {
	svc := &Service{
		organizations: make(map[string]*organizationsv1.Organization),
		members:       make(map[string]map[string]*organizationsv1.OrganizationMember),
	}

	svc.bootstrap()
	return svc
}

func (s *Service) ListOrganizations(_ context.Context, _ *connect.Request[organizationsv1.ListOrganizationsRequest]) (*connect.Response[organizationsv1.ListOrganizationsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	organizations := make([]*organizationsv1.Organization, 0, len(s.organizations))
	for _, org := range s.organizations {
		organizations = append(organizations, cloneOrganization(org))
	}

	sort.Slice(organizations, func(i, j int) bool {
		return organizations[i].GetId() < organizations[j].GetId()
	})

	pagination := &organizationsv1.Pagination{
		Page:       1,
		PerPage:    int32(len(organizations)),
		Total:      int32(len(organizations)),
		TotalPages: 1,
	}

	res := connect.NewResponse(&organizationsv1.ListOrganizationsResponse{
		Organizations: organizations,
		Pagination:    pagination,
	})
	return res, nil
}

func (s *Service) CreateOrganization(_ context.Context, req *connect.Request[organizationsv1.CreateOrganizationRequest]) (*connect.Response[organizationsv1.CreateOrganizationResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization name is required"))
	}

	plan := req.Msg.GetPlan()
	if plan == "" {
		plan = "starter"
	}

	slug := req.Msg.GetSlug()
	if slug == "" {
		slug = normalizeSlug(name)
	}

	domain := fmt.Sprintf("%s.obiente.cloud", slug)

	maxDeployments, maxVps, maxMembers := planLimits(plan)

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextOrganizationID()
	org := &organizationsv1.Organization{
		Id:              id,
		Name:            name,
		Slug:            slug,
		Plan:            strings.ToLower(plan),
		Status:          "active",
		CreatedAt:       timestamppb.Now(),
		MaxDeployments:  maxDeployments,
		MaxVpsInstances: maxVps,
		MaxTeamMembers:  maxMembers,
	}

	if domain != "" {
		org.Domain = &domain
	}

	s.organizations[id] = org
	s.members[id] = make(map[string]*organizationsv1.OrganizationMember)

	res := connect.NewResponse(&organizationsv1.CreateOrganizationResponse{Organization: cloneOrganization(org)})
	return res, nil
}

func (s *Service) GetOrganization(_ context.Context, req *connect.Request[organizationsv1.GetOrganizationRequest]) (*connect.Response[organizationsv1.GetOrganizationResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	org, ok := s.organizations[req.Msg.GetOrganizationId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization %s not found", req.Msg.GetOrganizationId()))
	}

	res := connect.NewResponse(&organizationsv1.GetOrganizationResponse{Organization: cloneOrganization(org)})
	return res, nil
}

func (s *Service) UpdateOrganization(_ context.Context, req *connect.Request[organizationsv1.UpdateOrganizationRequest]) (*connect.Response[organizationsv1.UpdateOrganizationResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	org, ok := s.organizations[req.Msg.GetOrganizationId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization %s not found", req.Msg.GetOrganizationId()))
	}

	if name := strings.TrimSpace(req.Msg.GetName()); name != "" {
		org.Name = name
	}

	if domain := req.Msg.GetDomain(); domain != "" {
		org.Domain = &domain
	}

	if req.Msg.GetDomain() == "" && req.Msg.Domain != nil {
		org.Domain = nil
	}

	res := connect.NewResponse(&organizationsv1.UpdateOrganizationResponse{Organization: cloneOrganization(org)})
	return res, nil
}

func (s *Service) ListMembers(_ context.Context, req *connect.Request[organizationsv1.ListMembersRequest]) (*connect.Response[organizationsv1.ListMembersResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, ok := s.members[req.Msg.GetOrganizationId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization %s not found", req.Msg.GetOrganizationId()))
	}

	list := make([]*organizationsv1.OrganizationMember, 0, len(members))
	for _, member := range members {
		list = append(list, cloneMember(member))
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].GetId() < list[j].GetId()
	})

	pagination := &organizationsv1.Pagination{
		Page:       1,
		PerPage:    int32(len(list)),
		Total:      int32(len(list)),
		TotalPages: 1,
	}

	res := connect.NewResponse(&organizationsv1.ListMembersResponse{
		Members:    list,
		Pagination: pagination,
	})
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

	s.mu.Lock()
	defer s.mu.Unlock()

	orgMembers, ok := s.members[req.Msg.GetOrganizationId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization %s not found", req.Msg.GetOrganizationId()))
	}

	id := s.nextMemberID()
	member := &organizationsv1.OrganizationMember{
		Id:     id,
		User:   &authv1.User{Id: fmt.Sprintf("user-%s", id), Email: email, Name: deriveNameFromEmail(email)},
		Role:   strings.ToLower(role),
		Status: "invited",
	}
	member.JoinedAt = timestamppb.Now()

	orgMembers[id] = member

	res := connect.NewResponse(&organizationsv1.InviteMemberResponse{Member: cloneMember(member)})
	return res, nil
}

func (s *Service) UpdateMember(_ context.Context, req *connect.Request[organizationsv1.UpdateMemberRequest]) (*connect.Response[organizationsv1.UpdateMemberResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orgMembers, ok := s.members[req.Msg.GetOrganizationId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization %s not found", req.Msg.GetOrganizationId()))
	}

	member, ok := orgMembers[req.Msg.GetMemberId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member %s not found", req.Msg.GetMemberId()))
	}

	if role := req.Msg.GetRole(); role != "" {
		member.Role = strings.ToLower(role)
	}

	member.Status = "active"

	res := connect.NewResponse(&organizationsv1.UpdateMemberResponse{Member: cloneMember(member)})
	return res, nil
}

func (s *Service) RemoveMember(_ context.Context, req *connect.Request[organizationsv1.RemoveMemberRequest]) (*connect.Response[organizationsv1.RemoveMemberResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orgMembers, ok := s.members[req.Msg.GetOrganizationId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization %s not found", req.Msg.GetOrganizationId()))
	}

	if _, ok := orgMembers[req.Msg.GetMemberId()]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("member %s not found", req.Msg.GetMemberId()))
	}

	delete(orgMembers, req.Msg.GetMemberId())

	res := connect.NewResponse(&organizationsv1.RemoveMemberResponse{Success: true})
	return res, nil
}

func (s *Service) bootstrap() {
	org := &organizationsv1.Organization{
		Id:              "org-001",
		Name:            "Obiente Cloud",
		Slug:            "obiente-cloud",
		Plan:            "pro",
		Status:          "active",
		CreatedAt:       timestamppb.New(time.Now().Add(-720 * time.Hour)),
		MaxDeployments:  25,
		MaxVpsInstances: 10,
		MaxTeamMembers:  50,
	}
	domain := "obiente.cloud"
	org.Domain = &domain

	s.organizations[org.Id] = org
	s.organizationID = 1

	member := &organizationsv1.OrganizationMember{
		Id: "mem-001",
		User: &authv1.User{
			Id:        "user_mock_123",
			Email:     "developer@obiente.cloud",
			Name:      "Obiente Developer",
			AvatarUrl: "https://cdn.obiente.cloud/assets/avatar/mock.png",
		},
		Role:     "owner",
		Status:   "active",
		JoinedAt: timestamppb.New(time.Now().Add(-700 * time.Hour)),
	}

	s.members[org.Id] = map[string]*organizationsv1.OrganizationMember{member.Id: member}
	s.memberID = 1
}

func (s *Service) nextOrganizationID() string {
	s.organizationID++
	return fmt.Sprintf("org-%03d", s.organizationID)
}

func (s *Service) nextMemberID() string {
	s.memberID++
	return fmt.Sprintf("mem-%03d", s.memberID)
}

func cloneOrganization(src *organizationsv1.Organization) *organizationsv1.Organization {
	if src == nil {
		return nil
	}
	out := &organizationsv1.Organization{
		Id:              src.GetId(),
		Name:            src.GetName(),
		Slug:            src.GetSlug(),
		Plan:            src.GetPlan(),
		Status:          src.GetStatus(),
		MaxDeployments:  src.GetMaxDeployments(),
		MaxVpsInstances: src.GetMaxVpsInstances(),
		MaxTeamMembers:  src.GetMaxTeamMembers(),
	}
	if d := src.GetDomain(); d != "" {
		out.Domain = &d
	}
	if ts := src.GetCreatedAt(); ts != nil {
		out.CreatedAt = timestamppb.New(ts.AsTime())
	}
	return out
}

func cloneMember(src *organizationsv1.OrganizationMember) *organizationsv1.OrganizationMember {
	if src == nil {
		return nil
	}
	out := &organizationsv1.OrganizationMember{
		Id:     src.GetId(),
		Role:   src.GetRole(),
		Status: src.GetStatus(),
	}
	if u := src.GetUser(); u != nil {
		out.User = cloneUser(u)
	}
	if ts := src.GetJoinedAt(); ts != nil {
		out.JoinedAt = timestamppb.New(ts.AsTime())
	}
	return out
}

func cloneUser(src *authv1.User) *authv1.User {
	if src == nil {
		return nil
	}
	out := &authv1.User{
		Id:        src.GetId(),
		Email:     src.GetEmail(),
		Name:      src.GetName(),
		AvatarUrl: src.GetAvatarUrl(),
	}
	if ts := src.GetCreatedAt(); ts != nil {
		out.CreatedAt = timestamppb.New(ts.AsTime())
	}
	return out
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

func planLimits(plan string) (deployments, vps, members int32) {
	switch strings.ToLower(plan) {
	case "pro":
		return 25, 10, 50
	case "enterprise":
		return 200, 100, 500
	default:
		return 5, 2, 10
	}
}

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
