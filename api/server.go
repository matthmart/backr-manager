package api

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/agence-webup/backr/manager"
	"github.com/agence-webup/backr/manager/proto"
	"github.com/dgrijalva/jwt-go"
)

func NewServer(projectRepo manager.ProjectRepository, fileRepo manager.FileRepository, accountRepo manager.AccountRepository, authConfig manager.AuthConfig) proto.BackrApiServer {
	srv := server{
		ProjectRepo: projectRepo,
		FileRepo:    fileRepo,
		AccountRepo: accountRepo,
		Config:      authConfig,
	}
	return &srv
}

type server struct {
	ProjectRepo manager.ProjectRepository
	FileRepo    manager.FileRepository
	AccountRepo manager.AccountRepository
	Config      manager.AuthConfig
}

func (srv *server) GetProjects(ctx context.Context, req *proto.GetProjectsRequest) (*proto.ProjectsListResponse, error) {
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

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
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

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
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

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
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

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
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

	file, err := srv.FileRepo.GetURL(manager.File{Path: req.Filepath})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get url: %v", err)
	}

	return &proto.GetFileURLResponse{Url: file.String()}, nil
}

func (srv *server) CreateAccount(ctx context.Context, req *proto.CreateAccountRequest) (*proto.AccountResponse, error) {
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

	if req.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username is required")
	}

	existingAccount, _ := srv.AccountRepo.Get(req.Username)
	if existingAccount != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "an account already exists with this username")
	}

	password, err := srv.AccountRepo.Create(req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to create account: %v", err)
	}

	return &proto.AccountResponse{
		Account:  &proto.Account{Username: req.Username},
		Password: password,
	}, nil
}

func (srv *server) ListAccounts(ctx context.Context, req *proto.ListAccountsRequest) (*proto.AccountsListResponse, error) {
	err := srv.authenticateRequest(ctx)
	if err != nil {
		return nil, err
	}

	rawAccounts, err := srv.AccountRepo.List()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to fetch account list: %v", err)
	}

	accounts := []*proto.Account{}
	for _, a := range rawAccounts {
		accounts = append(accounts, &proto.Account{Username: a.Username})
	}

	return &proto.AccountsListResponse{Accounts: accounts}, nil
}

func (srv *server) AuthenticateAccount(ctx context.Context, req *proto.AuthenticateAccountRequest) (*proto.AuthenticateAccountResponse, error) {

	err := srv.AccountRepo.Authenticate(req.Username, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unable to authenticate: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    "backr-manager",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Unix(),
		Subject:   req.Username,
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
		Audience:  "backr-manager-api",
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(srv.Config.JWTSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to create a JWT token: %v", err)
	}

	return &proto.AuthenticateAccountResponse{Token: tokenString}, nil
}

func transformToProtoProject(project manager.Project) proto.Project {
	rules := []*proto.Rule{}
	for _, r := range project.Rules {
		rule := proto.Rule{MinAge: int32(r.MinAge), Count: int32(r.Count)}

		if state, ok := project.State[r.GetID()]; ok {
			rule.Error = transformToProtoError(state.Error)
			if state.Next != nil {
				rule.NextDate = state.Next.Unix()
			}

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
