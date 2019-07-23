package inmem

import (
	"time"

	"github.com/agence-webup/backr/manager"
)

// NewProjectRepository returns an instance of
// an in-memory Project Repository
func NewProjectRepository() manager.ProjectRepository {
	r := projectRepo{
		projectsByName: map[string]manager.Project{},
	}
	return &r
}

type projectRepo struct {
	projectsByName map[string]manager.Project
}

func (repo *projectRepo) GetAll() ([]manager.Project, error) {
	projects := []manager.Project{}
	for _, p := range repo.projectsByName {
		projects = append(projects, p)
	}
	return projects, nil
}

func (repo *projectRepo) GetByName(name string) (*manager.Project, error) {
	p, ok := repo.projectsByName[name]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (repo *projectRepo) Save(project manager.Project) error {

	// set created at if needed
	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
	}

	repo.projectsByName[project.Name] = project

	return nil
}
