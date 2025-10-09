import { describe, it, expect, beforeEach } from 'vitest';
import { ConnectRouter } from '@connectrpc/connect';
import { OrganizationService } from '../../src/generated/obiente/cloud/organizations/v1/organization_service_connect';
import {
  ListOrganizationsRequest,
  ListOrganizationsResponse,
  CreateOrganizationRequest,
  CreateOrganizationResponse,
  GetOrganizationRequest,
  GetOrganizationResponse,
  UpdateOrganizationRequest,
  UpdateOrganizationResponse,
  ListMembersRequest,
  ListMembersResponse,
  InviteMemberRequest,
  InviteMemberResponse,
  UpdateMemberRequest,
  UpdateMemberResponse,
  RemoveMemberRequest,
  RemoveMemberResponse,
  Organization,
  OrganizationMember,
  Pagination,
} from '../../src/generated/obiente/cloud/organizations/v1/organization_service_pb';

/**
 * Contract tests for OrganizationService protobuf interface
 * These tests validate CRUD operations for organizations and members
 * Tests should FAIL initially until OrganizationService is implemented
 */
describe('OrganizationService Contract Tests', () => {
  let organizationService: OrganizationService;
  const mockUserId = 'user_123';
  const mockOrgId = 'org_456';

  beforeEach(() => {
    // This will fail until OrganizationService is properly implemented
    organizationService = new OrganizationService();
  });

  describe('ListOrganizations', () => {
    it('should accept ListOrganizationsRequest and return ListOrganizationsResponse', async () => {
      const request = new ListOrganizationsRequest({
        page: 1,
        perPage: 10,
      });

      // This will fail - no implementation yet
      const response = await organizationService.listOrganizations(request);

      expect(response).toBeInstanceOf(ListOrganizationsResponse);
      expect(response.organizations).toBeInstanceOf(Array);
      expect(response.pagination).toBeInstanceOf(Pagination);
      expect(response.pagination.page).toBe(1);
      expect(response.pagination.perPage).toBe(10);
      expect(response.pagination.total).toBeGreaterThanOrEqual(0);
    });

    it('should handle pagination correctly', async () => {
      const request = new ListOrganizationsRequest({
        page: 2,
        perPage: 5,
      });

      const response = await organizationService.listOrganizations(request);

      expect(response.pagination.page).toBe(2);
      expect(response.pagination.perPage).toBe(5);
      expect(response.organizations.length).toBeLessThanOrEqual(5);
    });

    it('should only return organizations user has access to', async () => {
      const request = new ListOrganizationsRequest({
        page: 1,
        perPage: 50,
      });

      const response = await organizationService.listOrganizations(request);

      // Each organization should have the user as a member
      for (const org of response.organizations) {
        expect(org.id).toBeDefined();
        expect(org.name).toBeDefined();
        expect(org.slug).toBeDefined();
        expect(['starter', 'pro', 'enterprise']).toContain(org.plan);
        expect(['active', 'suspended', 'trial']).toContain(org.status);
      }
    });
  });

  describe('CreateOrganization', () => {
    it('should accept CreateOrganizationRequest and return CreateOrganizationResponse', async () => {
      const request = new CreateOrganizationRequest({
        name: 'Test Company',
        slug: 'test-company',
        plan: 'starter',
      });

      // This will fail - no implementation yet
      const response = await organizationService.createOrganization(request);

      expect(response).toBeInstanceOf(CreateOrganizationResponse);
      expect(response.organization).toBeInstanceOf(Organization);
      expect(response.organization.name).toBe('Test Company');
      expect(response.organization.slug).toBe('test-company');
      expect(response.organization.plan).toBe('starter');
      expect(response.organization.status).toBe('active');
      expect(response.organization.id).toBeDefined();
      expect(response.organization.createdAt).toBeDefined();
    });

    it('should validate organization name requirements', async () => {
      const request = new CreateOrganizationRequest({
        name: 'T', // Too short
        slug: 'test-company',
        plan: 'starter',
      });

      await expect(organizationService.createOrganization(request)).rejects.toThrow(/name/i);
    });

    it('should validate slug uniqueness', async () => {
      const request1 = new CreateOrganizationRequest({
        name: 'Company One',
        slug: 'duplicate-slug',
        plan: 'starter',
      });

      const request2 = new CreateOrganizationRequest({
        name: 'Company Two',
        slug: 'duplicate-slug',
        plan: 'pro',
      });

      await organizationService.createOrganization(request1);
      await expect(organizationService.createOrganization(request2)).rejects.toThrow(/slug/i);
    });

    it('should validate slug format', async () => {
      const request = new CreateOrganizationRequest({
        name: 'Test Company',
        slug: 'Invalid Slug!', // Contains spaces and special chars
        plan: 'starter',
      });

      await expect(organizationService.createOrganization(request)).rejects.toThrow(/slug/i);
    });

    it('should validate plan type', async () => {
      const request = new CreateOrganizationRequest({
        name: 'Test Company',
        slug: 'test-company',
        plan: 'invalid-plan',
      });

      await expect(organizationService.createOrganization(request)).rejects.toThrow(/plan/i);
    });

    it('should set creator as owner automatically', async () => {
      const request = new CreateOrganizationRequest({
        name: 'Test Company',
        slug: 'test-company',
        plan: 'starter',
      });

      const response = await organizationService.createOrganization(request);

      // Should create membership record with owner role
      const membersRequest = new ListMembersRequest({
        organizationId: response.organization.id,
        page: 1,
        perPage: 10,
      });

      const membersResponse = await organizationService.listMembers(membersRequest);
      
      expect(membersResponse.members).toHaveLength(1);
      expect(membersResponse.members[0].role).toBe('owner');
      expect(membersResponse.members[0].status).toBe('active');
    });
  });

  describe('GetOrganization', () => {
    it('should accept GetOrganizationRequest and return GetOrganizationResponse', async () => {
      const request = new GetOrganizationRequest({
        organizationId: mockOrgId,
      });

      // This will fail - no implementation yet
      const response = await organizationService.getOrganization(request);

      expect(response).toBeInstanceOf(GetOrganizationResponse);
      expect(response.organization).toBeInstanceOf(Organization);
      expect(response.organization.id).toBe(mockOrgId);
      expect(response.organization.name).toBeDefined();
      expect(response.organization.slug).toBeDefined();
    });

    it('should reject access to organizations user is not member of', async () => {
      const request = new GetOrganizationRequest({
        organizationId: 'unauthorized-org-id',
      });

      await expect(organizationService.getOrganization(request)).rejects.toThrow(/not found|unauthorized/i);
    });

    it('should return complete organization details', async () => {
      const request = new GetOrganizationRequest({
        organizationId: mockOrgId,
      });

      const response = await organizationService.getOrganization(request);

      expect(response.organization.maxDeployments).toBeGreaterThan(0);
      expect(response.organization.maxVpsInstances).toBeGreaterThan(0);
      expect(response.organization.maxTeamMembers).toBeGreaterThan(0);
    });
  });

  describe('UpdateOrganization', () => {
    it('should accept UpdateOrganizationRequest and return UpdateOrganizationResponse', async () => {
      const request = new UpdateOrganizationRequest({
        organizationId: mockOrgId,
        name: 'Updated Company Name',
        domain: 'updated-company.com',
      });

      // This will fail - no implementation yet
      const response = await organizationService.updateOrganization(request);

      expect(response).toBeInstanceOf(UpdateOrganizationResponse);
      expect(response.organization.name).toBe('Updated Company Name');
      expect(response.organization.domain).toBe('updated-company.com');
    });

    it('should require admin or owner permissions', async () => {
      const request = new UpdateOrganizationRequest({
        organizationId: mockOrgId,
        name: 'Unauthorized Update',
      });

      // Mock user with member role (not admin/owner)
      await expect(organizationService.updateOrganization(request)).rejects.toThrow(/permission/i);
    });

    it('should validate domain format when provided', async () => {
      const request = new UpdateOrganizationRequest({
        organizationId: mockOrgId,
        domain: 'invalid-domain-format',
      });

      await expect(organizationService.updateOrganization(request)).rejects.toThrow(/domain/i);
    });
  });

  describe('ListMembers', () => {
    it('should accept ListMembersRequest and return ListMembersResponse', async () => {
      const request = new ListMembersRequest({
        organizationId: mockOrgId,
        page: 1,
        perPage: 10,
      });

      // This will fail - no implementation yet
      const response = await organizationService.listMembers(request);

      expect(response).toBeInstanceOf(ListMembersResponse);
      expect(response.members).toBeInstanceOf(Array);
      expect(response.pagination).toBeInstanceOf(Pagination);
    });

    it('should return members with complete user information', async () => {
      const request = new ListMembersRequest({
        organizationId: mockOrgId,
        page: 1,
        perPage: 10,
      });

      const response = await organizationService.listMembers(request);

      for (const member of response.members) {
        expect(member.id).toBeDefined();
        expect(member.user).toBeDefined();
        expect(member.user.email).toMatch(/^[^\s@]+@[^\s@]+\.[^\s@]+$/);
        expect(['owner', 'admin', 'member', 'viewer']).toContain(member.role);
        expect(['active', 'invited', 'suspended']).toContain(member.status);
      }
    });

    it('should respect organization membership for access', async () => {
      const request = new ListMembersRequest({
        organizationId: 'unauthorized-org',
        page: 1,
        perPage: 10,
      });

      await expect(organizationService.listMembers(request)).rejects.toThrow(/not found|unauthorized/i);
    });
  });

  describe('InviteMember', () => {
    it('should accept InviteMemberRequest and return InviteMemberResponse', async () => {
      const request = new InviteMemberRequest({
        organizationId: mockOrgId,
        email: 'newmember@example.com',
        role: 'member',
      });

      // This will fail - no implementation yet
      const response = await organizationService.inviteMember(request);

      expect(response).toBeInstanceOf(InviteMemberResponse);
      expect(response.member).toBeInstanceOf(OrganizationMember);
      expect(response.member.user.email).toBe('newmember@example.com');
      expect(response.member.role).toBe('member');
      expect(response.member.status).toBe('invited');
    });

    it('should require admin or owner permissions to invite', async () => {
      const request = new InviteMemberRequest({
        organizationId: mockOrgId,
        email: 'newmember@example.com',
        role: 'member',
      });

      // Mock user with viewer role
      await expect(organizationService.inviteMember(request)).rejects.toThrow(/permission/i);
    });

    it('should validate email format', async () => {
      const request = new InviteMemberRequest({
        organizationId: mockOrgId,
        email: 'invalid-email',
        role: 'member',
      });

      await expect(organizationService.inviteMember(request)).rejects.toThrow(/email/i);
    });

    it('should validate role is allowed', async () => {
      const request = new InviteMemberRequest({
        organizationId: mockOrgId,
        email: 'newmember@example.com',
        role: 'invalid-role',
      });

      await expect(organizationService.inviteMember(request)).rejects.toThrow(/role/i);
    });

    it('should prevent duplicate invitations', async () => {
      const request = new InviteMemberRequest({
        organizationId: mockOrgId,
        email: 'existing@example.com',
        role: 'member',
      });

      await organizationService.inviteMember(request);
      await expect(organizationService.inviteMember(request)).rejects.toThrow(/already/i);
    });

    it('should respect organization team member limits', async () => {
      // Mock organization with team limit reached
      const request = new InviteMemberRequest({
        organizationId: mockOrgId,
        email: 'overlimit@example.com',
        role: 'member',
      });

      await expect(organizationService.inviteMember(request)).rejects.toThrow(/limit/i);
    });
  });

  describe('UpdateMember', () => {
    it('should accept UpdateMemberRequest and return UpdateMemberResponse', async () => {
      const request = new UpdateMemberRequest({
        organizationId: mockOrgId,
        memberId: 'member_789',
        role: 'admin',
      });

      // This will fail - no implementation yet
      const response = await organizationService.updateMember(request);

      expect(response).toBeInstanceOf(UpdateMemberResponse);
      expect(response.member.role).toBe('admin');
    });

    it('should require admin or owner permissions', async () => {
      const request = new UpdateMemberRequest({
        organizationId: mockOrgId,
        memberId: 'member_789',
        role: 'admin',
      });

      await expect(organizationService.updateMember(request)).rejects.toThrow(/permission/i);
    });

    it('should prevent changing owner role', async () => {
      const request = new UpdateMemberRequest({
        organizationId: mockOrgId,
        memberId: 'owner_member_id',
        role: 'member',
      });

      await expect(organizationService.updateMember(request)).rejects.toThrow(/owner/i);
    });
  });

  describe('RemoveMember', () => {
    it('should accept RemoveMemberRequest and return RemoveMemberResponse', async () => {
      const request = new RemoveMemberRequest({
        organizationId: mockOrgId,
        memberId: 'member_789',
      });

      // This will fail - no implementation yet
      const response = await organizationService.removeMember(request);

      expect(response).toBeInstanceOf(RemoveMemberResponse);
      expect(response.success).toBe(true);
    });

    it('should require admin or owner permissions', async () => {
      const request = new RemoveMemberRequest({
        organizationId: mockOrgId,
        memberId: 'member_789',
      });

      await expect(organizationService.removeMember(request)).rejects.toThrow(/permission/i);
    });

    it('should prevent removing the last owner', async () => {
      const request = new RemoveMemberRequest({
        organizationId: mockOrgId,
        memberId: 'only_owner_id',
      });

      await expect(organizationService.removeMember(request)).rejects.toThrow(/owner/i);
    });
  });

  describe('Multi-tenant Security', () => {
    it('should enforce organization-scoped data access', async () => {
      const org1Request = new GetOrganizationRequest({
        organizationId: 'org_1',
      });

      const org2Request = new GetOrganizationRequest({
        organizationId: 'org_2',
      });

      // User should only access their organization
      const org1Response = await organizationService.getOrganization(org1Request);
      expect(org1Response.organization.id).toBe('org_1');

      await expect(organizationService.getOrganization(org2Request)).rejects.toThrow();
    });

    it('should prevent cross-tenant member management', async () => {
      const request = new InviteMemberRequest({
        organizationId: 'other_org_id',
        email: 'hacker@example.com',
        role: 'admin',
      });

      await expect(organizationService.inviteMember(request)).rejects.toThrow(/not found|unauthorized/i);
    });
  });

  describe('Data Validation', () => {
    it('should sanitize input data', async () => {
      const request = new CreateOrganizationRequest({
        name: '<script>alert("xss")</script>Company',
        slug: 'clean-slug',
        plan: 'starter',
      });

      const response = await organizationService.createOrganization(request);

      // Name should be sanitized
      expect(response.organization.name).not.toContain('<script>');
      expect(response.organization.name).toContain('Company');
    });

    it('should validate required fields', async () => {
      const request = new CreateOrganizationRequest({
        name: '',
        slug: '',
        plan: '',
      });

      await expect(organizationService.createOrganization(request)).rejects.toThrow(/required/i);
    });
  });
});