package inmem

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/agence-webup/backr/manager"
)

var sharedRepo projectRepo

// NewProjectRepository returns an instance of
// an in-memory Project Repository
func NewProjectRepository() manager.ProjectRepository {
	sharedRepo = projectRepo{
		filepath: "db.json",
	}

	sharedRepo.projectsByName = sharedRepo.readFromFile()

	return &sharedRepo
}

type projectRepo struct {
	filepath       string
	projectsByName map[string]manager.Project
}

func (repo *projectRepo) readFromFile() map[string]manager.Project {
	projectsByName := map[string]manager.Project{}
	if _, err := os.Stat(repo.filepath); os.IsNotExist(err) {
		return projectsByName
	}
	f, err := os.OpenFile(repo.filepath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Fatal("project repo: unable to read file:", err)
	}
	err = json.NewDecoder(f).Decode(&projectsByName)
	if err != nil {
		log.Fatal("project repo: unable to unmarshal data:", err)
	}
	return projectsByName
}

func (repo *projectRepo) saveToFile() {
	j, err := json.Marshal(repo.projectsByName)
	if err != nil {
		log.Fatal("project repo: unable to save to file:", err)
	}
	ioutil.WriteFile(repo.filepath, j, os.ModePerm)
}

func (repo *projectRepo) GetAll() ([]manager.Project, error) {
	projects := []manager.Project{}
	for _, p := range repo.projectsByName {
		projects = append(projects, p)
	}
	return projects, nil
}

func (repo *projectRepo) Save(project manager.Project) error {
	repo.projectsByName[project.Name] = project
	repo.saveToFile()
	return nil
}
