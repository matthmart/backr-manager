package s3

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/agence-webup/backr/manager"

	"github.com/minio/minio-go"
	"github.com/rs/zerolog/log"
)

// NewFileRepository returns an implementation of FileRepository using an S3 object storage
func NewFileRepository(config manager.S3Config) (manager.FileRepository, error) {
	minioClient, err := minio.New(config.Endpoint, config.AccessKey, config.SecretKey, config.UseTLS)
	if err != nil {
		return nil, err
	}

	sharedClient := fileRepository{
		bucket:      config.Bucket,
		minioClient: minioClient,
	}

	return &sharedClient, nil
}

type fileRepository struct {
	bucket      string
	minioClient *minio.Client
}

func (repo *fileRepository) GetAll() ([]manager.File, error) {

	// Create a done channel.
	doneCh := make(chan struct{})
	defer close(doneCh)

	// Recursively list all objects
	recursive := true

	log.Debug().Str("bucket", repo.bucket).Msg("fetching files in S3")

	files := []manager.File{}
	for object := range repo.minioClient.ListObjectsV2(repo.bucket, "", recursive, doneCh) {
		f := manager.File{
			Path: object.Key,
			Date: object.LastModified,
			Size: object.Size,
		}

		// getFileComponents checks for path conformance
		_, err := repo.getFileComponents(f)
		if err == nil {
			files = append(files, f)
		}
	}

	return files, nil
}

func (repo *fileRepository) GetAllByFolder() (manager.FilesByFolder, error) {

	filesByFolder := manager.FilesByFolder{}

	files, err := repo.GetAll()
	if err != nil {
		return nil, err
	}

	for _, f := range files {
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

func (repo *fileRepository) GetFolderForFile(file manager.File) (string, error) {
	components, err := repo.getFileComponents(file)
	if err != nil {
		return "", err
	}

	return components[0], nil
}

func (repo *fileRepository) GetFilenameForFile(file manager.File) (string, error) {
	components, err := repo.getFileComponents(file)
	if err != nil {
		return "", err
	}

	return components[1], nil
}

func (repo *fileRepository) RemoveFile(file manager.File) error {
	return repo.minioClient.RemoveObject(repo.bucket, file.Path)
}

func (repo *fileRepository) GetURL(file manager.File) (*url.URL, error) {
	// Set request parameters for content-disposition.
	reqParams := make(url.Values)

	// Generates a presigned url which expires in a day.
	presignedURL, err := repo.minioClient.PresignedGetObject(repo.bucket, file.Path, 15*time.Minute, reqParams)
	if err != nil {
		return nil, err
	}

	return presignedURL, nil
}

func (repo *fileRepository) getFileComponents(file manager.File) ([]string, error) {
	components := strings.Split(file.Path, "/")

	// ensure that there is no more than 2 levels (bucket/folder/files)
	if len(components) != 2 {
		return nil, errors.New("files must be stored on 2 levels: folder/filename.ext")
	}

	return components, nil
}
