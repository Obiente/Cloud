package deployments

import (
	"context"
	"fmt"
	"sync"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler

	mu           sync.RWMutex
	deployments  map[string]*deploymentsv1.Deployment
	deploymentID int
}

func NewService() deploymentsv1connect.DeploymentServiceHandler {
	svc := &Service{
		deployments: make(map[string]*deploymentsv1.Deployment),
	}

	svc.bootstrap()
	return svc
}

func (s *Service) ListDeployments(_ context.Context, _ *connect.Request[deploymentsv1.ListDeploymentsRequest]) (*connect.Response[deploymentsv1.ListDeploymentsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*deploymentsv1.Deployment, 0, len(s.deployments))
	for _, deployment := range s.deployments {
		items = append(items, cloneDeployment(deployment))
	}

	res := connect.NewResponse(&deploymentsv1.ListDeploymentsResponse{
		Deployments: items,
		Pagination: &organizationsv1.Pagination{
			Page:       1,
			PerPage:    int32(len(items)),
			Total:      int32(len(items)),
			TotalPages: 1,
		},
	})
	return res, nil
}

func (s *Service) CreateDeployment(_ context.Context, req *connect.Request[deploymentsv1.CreateDeploymentRequest]) (*connect.Response[deploymentsv1.CreateDeploymentResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID()
	deployment := &deploymentsv1.Deployment{
		Id:             id,
		Name:           req.Msg.GetName(),
		Domain:         fmt.Sprintf("%s.obiente.cloud", req.Msg.GetName()),
		CustomDomains:  []string{fmt.Sprintf("app.%s.obiente.cloud", req.Msg.GetName())},
		Type:           req.Msg.GetType(),
		Branch:         req.Msg.GetBranch(),
		Status:         "created",
		HealthStatus:   "pending",
		LastDeployedAt: timestamppb.Now(),
		BandwidthUsage: 0,
		StorageUsage:   0,
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

	s.deployments[id] = deployment

	res := connect.NewResponse(&deploymentsv1.CreateDeploymentResponse{Deployment: cloneDeployment(deployment)})
	return res, nil
}

func (s *Service) GetDeployment(_ context.Context, req *connect.Request[deploymentsv1.GetDeploymentRequest]) (*connect.Response[deploymentsv1.GetDeploymentResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployment, ok := s.deployments[req.Msg.GetDeploymentId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	res := connect.NewResponse(&deploymentsv1.GetDeploymentResponse{Deployment: cloneDeployment(deployment)})
	return res, nil
}

func (s *Service) UpdateDeployment(_ context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deployment, ok := s.deployments[req.Msg.GetDeploymentId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	if req.Msg.Name != nil {
		deployment.Name = req.Msg.GetName()
	}
	if req.Msg.Branch != nil {
		deployment.Branch = req.Msg.GetBranch()
	}
	if req.Msg.BuildCommand != nil {
		if build := req.Msg.GetBuildCommand(); build != "" {
			deployment.BuildCommand = proto.String(build)
		} else {
			deployment.BuildCommand = nil
		}
	}
	if req.Msg.InstallCommand != nil {
		if install := req.Msg.GetInstallCommand(); install != "" {
			deployment.InstallCommand = proto.String(install)
		} else {
			deployment.InstallCommand = nil
		}
	}

	deployment.Status = "updated"
	deployment.HealthStatus = "pending"
	deployment.LastDeployedAt = timestamppb.Now()

	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentResponse{Deployment: cloneDeployment(deployment)})
	return res, nil
}

func (s *Service) TriggerDeployment(_ context.Context, req *connect.Request[deploymentsv1.TriggerDeploymentRequest]) (*connect.Response[deploymentsv1.TriggerDeploymentResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deployment, ok := s.deployments[req.Msg.GetDeploymentId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", req.Msg.GetDeploymentId()))
	}

	deployment.Status = "deploying"
	deployment.HealthStatus = "starting"

	res := connect.NewResponse(&deploymentsv1.TriggerDeploymentResponse{
		DeploymentId: deployment.GetId(),
		Status:       deployment.GetStatus(),
	})
	return res, nil
}

func (s *Service) StreamDeploymentStatus(_ context.Context, req *connect.Request[deploymentsv1.StreamDeploymentStatusRequest], stream *connect.ServerStream[deploymentsv1.DeploymentStatusUpdate]) error {
	updates := []deploymentsv1.DeploymentStatusUpdate{
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       "deploying",
			HealthStatus: "starting",
			Message:      proto.String("Build started"),
			Timestamp:    timestamppb.Now(),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       "deploying",
			HealthStatus: "verifying",
			Message:      proto.String("Running smoke tests"),
			Timestamp:    timestamppb.New(time.Now().Add(5 * time.Second)),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       "running",
			HealthStatus: "healthy",
			Message:      proto.String("Deployment complete"),
			Timestamp:    timestamppb.New(time.Now().Add(10 * time.Second)),
		},
	}

	for i := range updates {
		if err := stream.Send(&updates[i]); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetDeploymentLogs(_ context.Context, req *connect.Request[deploymentsv1.GetDeploymentLogsRequest]) (*connect.Response[deploymentsv1.GetDeploymentLogsResponse], error) {
	lines := req.Msg.GetLines()
	if lines <= 0 {
		lines = 5
	}

	logs := make([]string, 0, lines)
	for i := int32(0); i < lines; i++ {
		logs = append(logs, fmt.Sprintf("[%s] mock log line %d for deployment %s", time.Now().Format(time.RFC3339), i+1, req.Msg.GetDeploymentId()))
	}

	res := connect.NewResponse(&deploymentsv1.GetDeploymentLogsResponse{Logs: logs})
	return res, nil
}

func (s *Service) nextID() string {
	s.deploymentID++
	return fmt.Sprintf("deploy-%03d", s.deploymentID)
}

func (s *Service) bootstrap() {
	sample := &deploymentsv1.Deployment{
		Id:             "deploy-001",
		Name:           "dashboard",
		Domain:         "dashboard.obiente.cloud",
		CustomDomains:  []string{"app.dashboard.obiente.cloud"},
		Type:           "docker",
		Branch:         "main",
		Status:         "running",
		HealthStatus:   "healthy",
		LastDeployedAt: timestamppb.New(time.Now().Add(-2 * time.Hour)),
		BandwidthUsage: 512 * 1024 * 1024,
		StorageUsage:   10 * 1024 * 1024 * 1024,
		CreatedAt:      timestamppb.New(time.Now().Add(-240 * time.Hour)),
	}
	repo := "https://github.com/obiente/cloud"
	sample.RepositoryUrl = &repo

	s.deploymentID = 1
	s.deployments[sample.GetId()] = sample
}

func cloneDeployment(src *deploymentsv1.Deployment) *deploymentsv1.Deployment {
	if src == nil {
		return nil
	}
	// Avoid copying internal mutex fields by constructing a fresh message
	out := &deploymentsv1.Deployment{
		Id:             src.GetId(),
		Name:           src.GetName(),
		Domain:         src.GetDomain(),
		Type:           src.GetType(),
		Branch:         src.GetBranch(),
		Status:         src.GetStatus(),
		HealthStatus:   src.GetHealthStatus(),
		BandwidthUsage: src.GetBandwidthUsage(),
		StorageUsage:   src.GetStorageUsage(),
	}
	out.CustomDomains = append([]string(nil), src.GetCustomDomains()...)
	if src.RepositoryUrl != nil {
		repo := src.GetRepositoryUrl()
		out.RepositoryUrl = proto.String(repo)
	}
	if src.BuildCommand != nil {
		build := src.GetBuildCommand()
		out.BuildCommand = proto.String(build)
	}
	if src.InstallCommand != nil {
		install := src.GetInstallCommand()
		out.InstallCommand = proto.String(install)
	}
	if ts := src.GetLastDeployedAt(); ts != nil {
		out.LastDeployedAt = timestamppb.New(ts.AsTime())
	}
	if ts := src.GetCreatedAt(); ts != nil {
		out.CreatedAt = timestamppb.New(ts.AsTime())
	}
	return out
}
