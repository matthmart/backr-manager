package manager

// ProjectRepository defines methods required
// to work with projects
type ProjectRepository interface {
	GetAll() ([]Project, error)
	GetByName(name string) (*Project, error)
	Save(project Project) error
	Close() error
}

// FileRepository abstracts interactions
// with a file repository (e.g. S3, disk...)
type FileRepository interface {
	GetAll() ([]File, error)
	GetAllByFolder() (FilesByFolder, error)
	GetFolderForFile(File) (string, error)
	GetFilenameForFile(File) (string, error)
	RemoveFile(File) error
}
