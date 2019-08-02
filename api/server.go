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

func (srv *server) GetFiles(ctx context.Context, req *proto.GetFilesRequest) (*proto.GetFilesResponse, error) {
	if req.ProjectName != "" {
		filesByFolder, err := srv.FileRepo.GetAllByFolder()
		if err != nil {
			return nil, status.Error(codes.Internal, "unable to fetch files:"+err.Error())
		}

		rawFiles := filesByFolder[req.ProjectName]
		files := []*proto.File{}
		for _, rf := range rawFiles {
			f := transformToProtoFile(rf)
			files = append(files, &f)
		}

		return &proto.GetFilesResponse{
			Files: files,
		}, nil
	}

	// all files
	rawFiles, err := srv.FileRepo.GetAll()
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to fetch files:"+err.Error())
	}
	files := []*proto.File{}
	for _, rf := range rawFiles {
		f := transformToProtoFile(rf)
		files = append(files, &f)
	}

	return &proto.GetFilesResponse{
		Files: files,
	}, nil
}

func (srv *server) GetFileURL(ctx context.Context, req *proto.GetFileURLRequest) (*proto.GetFileURLResponse, error) {

	file, err := srv.FileRepo.GetURL(manager.File{Path: req.Filepath})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get url: %v", err)
	}

	return &proto.GetFileURLResponse{Url: file.String()}, nil
}

func transformToProtoProject(project manager.Project) proto.Project {
	rules := []*proto.Rule{}
	for _, r := range project.Rules {
		rule := proto.Rule{MinAge: int32(r.MinAge), Count: int32(r.Count)}

		if state, ok := project.State[r.GetID()]; ok {
			rule.Error = transformToProtoError(state.Error)

			files := []*proto.File{}
			for _, f := range state.Files {
				file := proto.File{Path: f.Path, Date: f.Date.Unix(), Size: f.Size, Expiration: f.Expiration.Unix()}
				file.Error = transformToProtoError(f.Error)
				files = append(files, &file)
			}
			rule.Files = files
		}

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

func transformToProtoFile(file manager.File) proto.File {
	f := proto.File{
		Path: file.Path,
		Date: file.Date.Unix(),
		Size: file.Size,
	}

	return f
}

func transformToProtoError(err *manager.RuleStateError) proto.Error {
	if err == nil {
		return proto.Error_NO_ERROR
	}

	switch err.Reason {
	case manager.RuleStateErrorObsolete:
		return proto.Error_OBSOLETE
	case manager.RuleStateErrorSizeTooSmall:
		return proto.Error_TOO_SMALL
	case manager.RuleStateErrorNoFile:
		return proto.Error_NO_FILE
	}

	return proto.Error_UNKNOWN
}
