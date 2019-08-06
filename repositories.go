package manager

import "net/url"

// ProjectRepository defines methods required
// to work with projects
type ProjectRepository interface {
	GetAll() ([]Project, error)
	GetByName(name string) (*Project, error)
	Save(project Project) error
}

// FileRepository abstracts interactions
// with a file repository (e.g. S3, disk...)
type FileRepository interface {
	GetAll() ([]File, error)
	GetAllByFolder() (FilesByFolder, error)
	GetFolderForFile(File) (string, error)
	GetFilenameForFile(File) (string, error)
	RemoveFile(File) error
	GetURL(File) (*url.URL, error)
}

// AccountRepository abstracts interactions with
// user accounts allowed to manage projects & files
type AccountRepository interface {
	List() ([]Account, error)
	Get(username string) (*Account, error)
	// Create must return an automatically generated password for the created user
	Create(username string) (string, error)
	Delete(username string) error
	ChangePassword(username string) (string, error)
	Authenticate(username, password string) error
}
