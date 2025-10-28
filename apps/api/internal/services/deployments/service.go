package deployments

import (
	"context"
	"fmt"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	"api/internal/database"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler
	repo *database.DeploymentRepository
}

func NewService(repo *database.DeploymentRepository) deploymentsv1connect.DeploymentServiceHandler {
	return &Service{
		repo: repo,
	}
}

func (s *Service) ListDeployments(ctx context.Context, req *connect.Request[deploymentsv1.ListDeploymentsRequest]) (*connect.Response[deploymentsv1.ListDeploymentsResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		orgID = "default"
	}

	filters := &database.DeploymentFilters{}
	if status := req.Msg.Status; status != nil {
		statusVal := int32(*status)
		filters.Status = &statusVal
	}

	dbDeployments, err := s.repo.GetAll(ctx, orgID, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list deployments: %w", err))
	}

	items := make([]*deploymentsv1.Deployment, 0, len(dbDeployments))
	for _, dbDep := range dbDeployments {
		items = append(items, dbDeploymentToProto(dbDep))
	}

	total, err := s.repo.Count(ctx, orgID)
	if err != nil {
		total = int64(len(dbDeployments))
	}

	res := connect.NewResponse(&deploymentsv1.ListDeploymentsResponse{
		Deployments: items,
		Pagination: &organizationsv1.Pagination{
			Page:       1,
			PerPage:    int32(len(items)),
			Total:      int32(total),
			TotalPages: 1,
		},
	})
	return res, nil
}

func (s *Service) CreateDeployment(ctx context.Context, req *connect.Request[deploymentsv1.CreateDeploymentRequest]) (*connect.Response[deploymentsv1.CreateDeploymentResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		orgID = "default"
	}

	// Generate ID
	id := fmt.Sprintf("deploy-%d", time.Now().Unix())

	// Create deployment in proto format first
	deployment := &deploymentsv1.Deployment{
		Id:             id,
		Name:           req.Msg.GetName(),
		Domain:         fmt.Sprintf("%s.obiente.cloud", req.Msg.GetName()),
		CustomDomains:  []string{fmt.Sprintf("app.%s.obiente.cloud", req.Msg.GetName())},
		Type:           req.Msg.GetType(),
		Branch:         req.Msg.GetBranch(),
		Status:         deploymentsv1.DeploymentStatus_CREATED,
		HealthStatus:   "pending",
		Environment:    deploymentsv1.Environment_PRODUCTION,
		LastDeployedAt: timestamppb.Now(),
		BandwidthUsage: 0,
		StorageUsage:   0,
		BuildTime:      0,
		Size:           "--",
		CreatedAt:      timestamppb.Now(),
	}

	if repo := req.Msg.GetRepositoryUrl(); repo != "" {
		deployment.RepositoryUrl = proto.String(repo)
	}
	if build := req.Msg.GetBuildCommand(); build != "" {
		deployment.BuildCommand = proto.String(build)
	}
	if install := req.Msg.GetInstallCommand(); install != "" {
		deployment.InstallCommand = proto.String(install)
	}

	// Convert to database model
	dbDeployment := protoToDBDeployment(deployment, orgID, "system")
	
	// Save to database
	if err := s.repo.Create(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	res := connect.NewResponse(&deploymentsv1.CreateDeploymentResponse{Deployment: deployment})
	return res, nil
}

func (s *Service) GetDeployment(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentRequest]) (*connect.Response[deploymentsv1.GetDeploymentResponse], error) {
	dbDeployment, err := s.repo.GetByID(ctx, req.Msg.GetDeploymentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	deployment := dbDeploymentToProto(dbDeployment)
	res := connect.NewResponse(&deploymentsv1.GetDeploymentResponse{Deployment: deployment})
	return res, nil
}

func (s *Service) UpdateDeployment(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentResponse], error) {
	dbDeployment, err := s.repo.GetByID(ctx, req.Msg.GetDeploymentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	if req.Msg.Name != nil {
		dbDeployment.Name = req.Msg.GetName()
	}
	if req.Msg.Branch != nil {
		dbDeployment.Branch = req.Msg.GetBranch()
	}
	if req.Msg.BuildCommand != nil {
		build := req.Msg.GetBuildCommand()
		if build != "" {
			dbDeployment.BuildCommand = &build
		} else {
			dbDeployment.BuildCommand = nil
		}
	}
	if req.Msg.InstallCommand != nil {
		install := req.Msg.GetInstallCommand()
		if install != "" {
			dbDeployment.InstallCommand = &install
		} else {
			dbDeployment.InstallCommand = nil
		}
	}

	dbDeployment.Status = int32(deploymentsv1.DeploymentStatus_BUILDING)
	dbDeployment.HealthStatus = "pending"
	dbDeployment.LastDeployedAt = time.Now()

	if err := s.repo.Update(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment: %w", err))
	}

	protoDeployment := dbDeploymentToProto(dbDeployment)
	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentResponse{Deployment: protoDeployment})
	return res, nil
}

func (s *Service) TriggerDeployment(ctx context.Context, req *connect.Request[deploymentsv1.TriggerDeploymentRequest]) (*connect.Response[deploymentsv1.TriggerDeploymentResponse], error) {
	if err := s.repo.UpdateStatus(ctx, req.Msg.GetDeploymentId(), int32(deploymentsv1.DeploymentStatus_DEPLOYING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deployment: %w", err))
	}

	// Simulate async deployment
	go func() {
		time.Sleep(10 * time.Second)
		s.repo.UpdateStatus(context.Background(), req.Msg.GetDeploymentId(), int32(deploymentsv1.DeploymentStatus_RUNNING))
	}()

	dbDeployment, _ := s.repo.GetByID(ctx, req.Msg.GetDeploymentId())
	res := connect.NewResponse(&deploymentsv1.TriggerDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Status:       "DEPLOYING",
	})
	if dbDeployment != nil {
		res.Msg.Status = getStatusName(dbDeployment.Status)
	}
	return res, nil
}

func (s *Service) StreamDeploymentStatus(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentStatusRequest], stream *connect.ServerStream[deploymentsv1.DeploymentStatusUpdate]) error {
	updates := []deploymentsv1.DeploymentStatusUpdate{
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_DEPLOYING,
			HealthStatus: "starting",
			Message:      proto.String("Build started"),
			Timestamp:    timestamppb.Now(),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_DEPLOYING,
			HealthStatus: "verifying",
			Message:      proto.String("Running smoke tests"),
			Timestamp:    timestamppb.New(time.Now().Add(5 * time.Second)),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_RUNNING,
			HealthStatus: "healthy",
			Message:      proto.String("Deployment complete"),
			Timestamp:    timestamppb.New(time.Now().Add(10 * time.Second)),
		},
	}

	for i := range updates {
		if err := stream.Send(&updates[i]); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (s *Service) GetDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentLogsRequest]) (*connect.Response[deploymentsv1.GetDeploymentLogsResponse], error) {
	lines := req.Msg.GetLines()
	if lines <= 0 {
		lines = 50
	}

	logs := make([]string, 0, lines)
	for i := int32(0); i < lines; i++ {
		logs = append(logs, fmt.Sprintf("[%s] Log line %d for deployment %s", time.Now().Format(time.RFC3339), i+1, req.Msg.GetDeploymentId()))
	}

	res := connect.NewResponse(&deploymentsv1.GetDeploymentLogsResponse{Logs: logs})
	return res, nil
}

func (s *Service) StartDeployment(ctx context.Context, req *connect.Request[deploymentsv1.StartDeploymentRequest]) (*connect.Response[deploymentsv1.StartDeploymentResponse], error) {
	dbDeployment, err := s.repo.GetByID(ctx, req.Msg.GetDeploymentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	if deploymentsv1.DeploymentStatus(dbDeployment.Status) != deploymentsv1.DeploymentStatus_STOPPED {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment must be stopped to start it"))
	}

	dbDeployment.Status = int32(deploymentsv1.DeploymentStatus_BUILDING)
	dbDeployment.LastDeployedAt = time.Now()

	if err := s.repo.Update(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start deployment: %w", err))
	}

	// Simulate async transition to RUNNING
	go func() {
		time.Sleep(2 * time.Second)
		s.repo.UpdateStatus(context.Background(), req.Msg.GetDeploymentId(), int32(deploymentsv1.DeploymentStatus_RUNNING))
	}()

	protoDeployment := dbDeploymentToProto(dbDeployment)
	res := connect.NewResponse(&deploymentsv1.StartDeploymentResponse{Deployment: protoDeployment})
	return res, nil
}

func (s *Service) StopDeployment(ctx context.Context, req *connect.Request[deploymentsv1.StopDeploymentRequest]) (*connect.Response[deploymentsv1.StopDeploymentResponse], error) {
	dbDeployment, err := s.repo.GetByID(ctx, req.Msg.GetDeploymentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	if deploymentsv1.DeploymentStatus(dbDeployment.Status) != deploymentsv1.DeploymentStatus_RUNNING {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment must be running to stop it"))
	}

	dbDeployment.Status = int32(deploymentsv1.DeploymentStatus_STOPPED)
	if err := s.repo.Update(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop deployment: %w", err))
	}

	protoDeployment := dbDeploymentToProto(dbDeployment)
	res := connect.NewResponse(&deploymentsv1.StopDeploymentResponse{Deployment: protoDeployment})
	return res, nil
}

func (s *Service) DeleteDeployment(ctx context.Context, req *connect.Request[deploymentsv1.DeleteDeploymentRequest]) (*connect.Response[deploymentsv1.DeleteDeploymentResponse], error) {
	if err := s.repo.Delete(ctx, req.Msg.GetDeploymentId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	res := connect.NewResponse(&deploymentsv1.DeleteDeploymentResponse{Success: true})
	return res, nil
}

func getStatusName(status int32) string {
	switch deploymentsv1.DeploymentStatus(status) {
	case deploymentsv1.DeploymentStatus_CREATED:
		return "CREATED"
	case deploymentsv1.DeploymentStatus_BUILDING:
		return "BUILDING"
	case deploymentsv1.DeploymentStatus_RUNNING:
		return "RUNNING"
	case deploymentsv1.DeploymentStatus_STOPPED:
		return "STOPPED"
	case deploymentsv1.DeploymentStatus_FAILED:
		return "FAILED"
	case deploymentsv1.DeploymentStatus_DEPLOYING:
		return "DEPLOYING"
	default:
		return "UNSPECIFIED"
	}
}

