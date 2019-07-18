package inmem

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agence-webup/backr/manager"
)

// var sharedFileRepo *fileRepo

type fileRepo struct {
	Files []manager.File
}

// NewFileRepository returns a FileRepository instance,
// simulating some files (storing fake data in memory).
//
// The main usage of this repository is unit testing.
func NewFileRepository() manager.FileRepository {
	// if sharedFileRepo == nil {
	// 	sharedFileRepo = new(fileRepo)
	// }
	// return sharedFileRepo
	return new(fileRepo)
}

func (repo *fileRepo) GetAll() ([]manager.File, error) {
	return repo.Files, nil
}

func (repo *fileRepo) GetAllByFolder() (manager.FilesByFolder, error) {
	filesByFolder := manager.FilesByFolder{}
	for _, f := range repo.Files {
		// get the file's folder
		folder, err := repo.GetFolderForFile(f)
		if err != nil {
			return nil, fmt.Errorf("unable to get folder for file '%v': %v", f.Path, err)
		}

		// check if some files already exists in this folder
		// if not, create an empty slice
		folderFiles, ok := filesByFolder[folder]
		if !ok {
			folderFiles = []manager.File{}
		}

		// append the file to the folder slice
		folderFiles = append(folderFiles, f)
		filesByFolder[folder] = folderFiles
	}

	return filesByFolder, nil
}

func (repo *fileRepo) GetFolderForFile(file manager.File) (string, error) {
	components, err := repo.getFileComponents(file)
	if err != nil {
		return "", err
	}

	return components[0], nil
}

func (repo *fileRepo) GetFilenameForFile(file manager.File) (string, error) {
	components, err := repo.getFileComponents(file)
	if err != nil {
		return "", err
	}

	return components[1], nil
}

func (repo *fileRepo) RemoveFile(file manager.File) error {
	fileIndex := 0
	for i, f := range repo.Files {
		if file.Path == f.Path {
			fileIndex = i
			break
		}
	}

	repo.Files = append(repo.Files[:fileIndex], repo.Files[fileIndex+1:]...)

	return nil
}

func (repo *fileRepo) getFileComponents(file manager.File) ([]string, error) {
	components := strings.Split(file.Path, "/")

	// ensure that there is no more than 2 levels (bucket/folder/files)
	if len(components) != 2 {
		return nil, errors.New("files must be stored on 2 levels: folder/filename.ext")
	}

	return components, nil
}

func CreateFakeFile(repo manager.FileRepository, file manager.File) {
	r, ok := repo.(*fileRepo)
	if !ok {
		return
	}
	r.Files = append(r.Files, file)
}
