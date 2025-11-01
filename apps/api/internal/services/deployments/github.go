package deployments

import (
	"context"
	"fmt"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"
	githubclient "api/internal/services/github"

	"connectrpc.com/connect"
)

// getGitHubToken retrieves a GitHub token for the authenticated user or organization
func (s *Service) getGitHubToken(ctx context.Context, orgID string, integrationID string) (string, error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("authentication required")
	}

	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)

	// First try specific integration ID if provided
	if integrationID != "" {
		var integration database.GitHubIntegration
		if err := database.DB.Where("id = ?", integrationID).First(&integration).Error; err == nil {
			// Verify user has access to this integration
			if integration.UserID != nil && *integration.UserID == user.Id {
				return integration.Token, nil
			}
			if integration.OrganizationID != nil {
				// Check if user is member of the organization
				if isSuperAdmin {
					return integration.Token, nil
				}
				var member database.OrganizationMember
				if err := database.DB.Where("organization_id = ? AND user_id = ?", *integration.OrganizationID, user.Id).First(&member).Error; err == nil {
					return integration.Token, nil
				}
			}
		}
	}

	// Then try organization token if orgID is provided
	if orgID != "" {
		var orgIntegration database.GitHubIntegration
		if err := database.DB.Where("organization_id = ?", orgID).First(&orgIntegration).Error; err == nil {
			if isSuperAdmin {
				return orgIntegration.Token, nil
			}
			var member database.OrganizationMember
			if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err == nil {
				return orgIntegration.Token, nil
			}
		}
	}

	// Fall back to user token
	var userIntegration database.GitHubIntegration
	if err := database.DB.Where("user_id = ?", user.Id).First(&userIntegration).Error; err == nil {
		return userIntegration.Token, nil
	}

	return "", fmt.Errorf("no GitHub integration found for user or organization")
}

// ListGitHubRepos lists GitHub repositories for the authenticated user or organization
func (s *Service) ListGitHubRepos(ctx context.Context, req *connect.Request[deploymentsv1.ListGitHubReposRequest]) (*connect.Response[deploymentsv1.ListGitHubReposResponse], error) {
	_, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	orgID := req.Msg.GetOrganizationId()
	integrationID := req.Msg.GetIntegrationId()
	ghToken, err := s.getGitHubToken(ctx, orgID, integrationID)
	if err != nil {
		// Return empty list if no GitHub integration is found
		return connect.NewResponse(&deploymentsv1.ListGitHubReposResponse{
			Repos: []*deploymentsv1.GitHubRepo{},
			Total: 0,
		}), nil
	}

	ghClient := githubclient.NewClient(ghToken)
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	repos, err := ghClient.ListRepos(ctx, page, perPage)
	if err != nil {
		// If GitHub API fails (e.g., invalid token), return empty list
		return connect.NewResponse(&deploymentsv1.ListGitHubReposResponse{
			Repos: []*deploymentsv1.GitHubRepo{},
			Total: 0,
		}), nil
	}

	protoRepos := make([]*deploymentsv1.GitHubRepo, 0, len(repos))
	for _, r := range repos {
		protoRepos = append(protoRepos, &deploymentsv1.GitHubRepo{
			Id:            fmt.Sprintf("%d", r.ID),
			Name:          r.Name,
			FullName:      r.FullName,
			Description:   r.Description,
			Url:           r.URL,
			IsPrivate:     r.IsPrivate,
			DefaultBranch: r.DefaultBranch,
		})
	}

	return connect.NewResponse(&deploymentsv1.ListGitHubReposResponse{
		Repos: protoRepos,
		Total: int32(len(protoRepos)),
	}), nil
}

// GetGitHubBranches lists branches for a GitHub repository
func (s *Service) GetGitHubBranches(ctx context.Context, req *connect.Request[deploymentsv1.GetGitHubBranchesRequest]) (*connect.Response[deploymentsv1.GetGitHubBranchesResponse], error) {
	_, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	repoFullName := req.Msg.GetRepoFullName()
	if repoFullName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repo_full_name is required"))
	}

	orgID := req.Msg.GetOrganizationId()
	integrationID := req.Msg.GetIntegrationId()
	ghToken, err := s.getGitHubToken(ctx, orgID, integrationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("GitHub integration not found: %w", err))
	}

	ghClient := githubclient.NewClient(ghToken)

	branches, err := ghClient.ListBranches(ctx, repoFullName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch branches: %w", err))
	}

	protoBranches := make([]*deploymentsv1.GitHubBranch, 0, len(branches))
	for i, b := range branches {
		protoBranches = append(protoBranches, &deploymentsv1.GitHubBranch{
			Name:      b.Name,
			IsDefault: i == 0, // First branch is often default
			Sha:       b.Commit.SHA,
		})
	}

	return connect.NewResponse(&deploymentsv1.GetGitHubBranchesResponse{
		Branches: protoBranches,
	}), nil
}

// GetGitHubFile retrieves a file from a GitHub repository
func (s *Service) GetGitHubFile(ctx context.Context, req *connect.Request[deploymentsv1.GetGitHubFileRequest]) (*connect.Response[deploymentsv1.GetGitHubFileResponse], error) {
	_, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	repoFullName := req.Msg.GetRepoFullName()
	branch := req.Msg.GetBranch()
	path := req.Msg.GetPath()

	if repoFullName == "" || branch == "" || path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repo_full_name, branch, and path are required"))
	}

	orgID := req.Msg.GetOrganizationId()
	integrationID := req.Msg.GetIntegrationId()
	ghToken, err := s.getGitHubToken(ctx, orgID, integrationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("GitHub integration not found: %w", err))
	}

	ghClient := githubclient.NewClient(ghToken)

	fileContent, err := ghClient.GetFile(ctx, repoFullName, branch, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to fetch file: %w", err))
	}

	return connect.NewResponse(&deploymentsv1.GetGitHubFileResponse{
		Content:  fileContent.Content,
		Encoding: fileContent.Encoding,
		Size:     fileContent.Size,
	}), nil
}

// ListAvailableGitHubIntegrations lists all GitHub integrations accessible to the user
func (s *Service) ListAvailableGitHubIntegrations(ctx context.Context, req *connect.Request[deploymentsv1.ListAvailableGitHubIntegrationsRequest]) (*connect.Response[deploymentsv1.ListAvailableGitHubIntegrationsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	orgID := req.Msg.GetOrganizationId()

	var integrations []database.GitHubIntegration

	// If orgID is provided, filter by that organization
	if orgID != "" {
		// Get user's GitHub integration
		var userIntegration database.GitHubIntegration
		if err := database.DB.Where("user_id = ?", user.Id).First(&userIntegration).Error; err == nil {
			integrations = append(integrations, userIntegration)
		}

		// Get organization's GitHub integration (if user is member)
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err == nil {
			var orgIntegration database.GitHubIntegration
			if err := database.DB.Where("organization_id = ?", orgID).First(&orgIntegration).Error; err == nil {
				integrations = append(integrations, orgIntegration)
			}
		}
	} else {
		// Get all integrations user has access to
		// User's personal integration
		var userIntegration database.GitHubIntegration
		if err := database.DB.Where("user_id = ?", user.Id).First(&userIntegration).Error; err == nil {
			integrations = append(integrations, userIntegration)
		}

		// All organization integrations where user is a member
		var orgMemberships []database.OrganizationMember
		database.DB.Where("user_id = ?", user.Id).Find(&orgMemberships)

		for _, membership := range orgMemberships {
			var orgIntegration database.GitHubIntegration
			if err := database.DB.Where("organization_id = ?", membership.OrganizationID).First(&orgIntegration).Error; err == nil {
				integrations = append(integrations, orgIntegration)
			}
		}
	}

	protoIntegrations := make([]*deploymentsv1.GitHubIntegrationOption, 0, len(integrations))
	for _, integration := range integrations {
		option := &deploymentsv1.GitHubIntegrationOption{
			Id:       integration.ID,
			Username: integration.Username,
		}

		if integration.UserID != nil {
			option.IsUser = true
		} else if integration.OrganizationID != nil {
			option.IsUser = false
			option.ObienteOrgId = *integration.OrganizationID

			// Get organization name
			var org database.Organization
			if err := database.DB.Where("id = ?", *integration.OrganizationID).First(&org).Error; err == nil {
				option.ObienteOrgName = org.Name
			}
		}

		protoIntegrations = append(protoIntegrations, option)
	}

	return connect.NewResponse(&deploymentsv1.ListAvailableGitHubIntegrationsResponse{
		Integrations: protoIntegrations,
	}), nil
}
