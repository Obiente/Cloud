package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"

	"connectrpc.com/connect"
)

// GetDomainVerificationToken retrieves or generates a verification token for a domain
func (s *Service) GetDomainVerificationToken(ctx context.Context, req *connect.Request[deploymentsv1.GetDomainVerificationTokenRequest]) (*connect.Response[deploymentsv1.GetDomainVerificationTokenResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	domain := req.Msg.GetDomain()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}

	// Get or create verification token (calls internal method)
	token, err := s.getDomainVerificationTokenInternal(ctx, deploymentID, domain)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get verification token: %w", err))
	}

	// Determine status from custom_domains
	status := "pending"
	if dbDeployment.CustomDomains != "" {
		var customDomains []string
		if err := json.Unmarshal([]byte(dbDeployment.CustomDomains), &customDomains); err == nil {
			for _, entry := range customDomains {
				parts := strings.Split(entry, ":")
				if len(parts) >= 2 && strings.ToLower(parts[0]) == strings.ToLower(domain) {
					if len(parts) >= 4 && parts[1] == "token" {
						status = parts[3]
					} else if len(parts) >= 2 && parts[1] == "verified" {
						status = "verified"
					}
					break
				}
			}
		}
	}

	txtRecordName := fmt.Sprintf("_obiente-verification.%s", domain)
	txtRecordValue := fmt.Sprintf("obiente-verification=%s", token)

	res := connect.NewResponse(&deploymentsv1.GetDomainVerificationTokenResponse{
		Domain:         domain,
		Token:          token,
		TxtRecordName:  txtRecordName,
		TxtRecordValue: txtRecordValue,
		Status:         status,
	})
	return res, nil
}

// VerifyDomainOwnership verifies domain ownership via DNS TXT record
func (s *Service) VerifyDomainOwnership(ctx context.Context, req *connect.Request[deploymentsv1.VerifyDomainOwnershipRequest]) (*connect.Response[deploymentsv1.VerifyDomainOwnershipResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	domain := req.Msg.GetDomain()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}

	// Perform verification (calls internal method)
	err = s.verifyDomainOwnershipInternal(ctx, deploymentID, domain)
	if err != nil {
		errMsg := err.Error()
		return connect.NewResponse(&deploymentsv1.VerifyDomainOwnershipResponse{
			Domain:   domain,
			Verified: false,
			Status:   "failed",
			Message:  &errMsg,
		}), nil
	}

	// Get updated status
	dbDeployment, _ = s.repo.GetByID(ctx, deploymentID)
	status := "verified"
	if dbDeployment.CustomDomains != "" {
		var customDomains []string
		if err := json.Unmarshal([]byte(dbDeployment.CustomDomains), &customDomains); err == nil {
			for _, entry := range customDomains {
				parts := strings.Split(entry, ":")
				if len(parts) >= 2 && strings.ToLower(parts[0]) == strings.ToLower(domain) {
					if len(parts) >= 4 && parts[1] == "token" {
						status = parts[3]
					} else if len(parts) >= 2 && parts[1] == "verified" {
						status = "verified"
					}
					break
				}
			}
		}
	}

	res := connect.NewResponse(&deploymentsv1.VerifyDomainOwnershipResponse{
		Domain:   domain,
		Verified: true,
		Status:   status,
	})
	return res, nil
}

