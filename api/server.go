package api

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/agence-webup/backr/manager"
	"github.com/agence-webup/backr/manager/proto"
)

func NewServer(projectRepo manager.ProjectRepository, fileRepo manager.FileRepository) proto.BackrApiServer {
	srv := server{
		ProjectRepo: projectRepo,
		FileRepo:    fileRepo,
	}
	return &srv
}

type server struct {
	ProjectRepo manager.ProjectRepository
	FileRepo    manager.FileRepository
}

func (srv *server) GetProjects(ctx context.Context, req *proto.GetProjectsRequest) (*proto.ProjectsListResponse, error) {
	rawProjects, err := srv.ProjectRepo.GetAll()
	if err != nil {
		return nil, err
	}

	projects := []*proto.Project{}
	for _, rawP := range rawProjects {
		p := transformToProtoProject(rawP)
		projects = append(projects, &p)
	}

	return &proto.ProjectsListResponse{
		Projects: projects,
		Total:    int32(len(projects)),
	}, nil
}

func (srv *server) GetProject(ctx context.Context, req *proto.GetProjectRequest) (*proto.ProjectResponse, error) {
	rawProject, err := srv.ProjectRepo.GetByName(req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to fetch project from repo")
	}
	if rawProject == nil {
		return nil, status.Error(codes.NotFound, "project not found")
	}

	p := transformToProtoProject(*rawProject)

	return &proto.ProjectResponse{Project: &p}, nil
}

func (srv *server) CreateProject(ctx context.Context, req *proto.CreateProjectRequest) (*proto.CreateProjectResponse, error) {

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "'name' is required")
	}
	if len(req.Rules) == 0 {
		return nil, status.Error(codes.InvalidArgument, "'rules' is required and must not be empty")
	}

	existingProject, err := srv.ProjectRepo.GetByName(req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to get project by name")
	}
	if existingProject != nil {
		return nil, status.Error(codes.FailedPrecondition, "a project with this name already exists")
	}

	rules := []manager.Rule{}
	for _, r := range req.Rules {
		minAge := 1
		if r.MinAge > 0 {
			minAge = int(r.MinAge)
		}
		count := 3
		if r.Count > 0 {
			count = int(r.Count)
		}
		rule := manager.Rule{MinAge: minAge, Count: count}
		rules = append(rules, rule)
	}

	project := manager.Project{
		Name:      req.Name,
		Rules:     rules,
		CreatedAt: time.Now(),
	}

	srv.ProjectRepo.Save(project)

	protoProject := transformToProtoProject(project)
	resp := proto.CreateProjectResponse{
		Project: &protoProject,
	}

	return &resp, nil
}

func transformToProtoProject(project manager.Project) proto.Project {
	rules := []*proto.Rule{}
	for _, r := range project.Rules {
		rule := proto.Rule{MinAge: int32(r.MinAge), Count: int32(r.Count)}
		rules = append(rules, &rule)
	}

	p := proto.Project{
		Name:        project.Name,
		Rules:       rules,
		CreatedAt:   project.CreatedAt.UTC().Unix(),
		IssuesCount: 0,
	}

	return p
}
