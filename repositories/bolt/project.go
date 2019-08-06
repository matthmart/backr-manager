package bolt

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/agence-webup/backr/manager"
	bolt "go.etcd.io/bbolt"
)

var projectBucket = []byte("projects")

// NewProjectRepository returns an instance of
// a Project Repository backed by a Bolt database.
// Close() should be called to terminate gracefully
func NewProjectRepository(db *bolt.DB) manager.ProjectRepository {
	r := projectRepo{
		db: db,
	}
	return &r
}

type projectRepo struct {
	db *bolt.DB
}

func (repo *projectRepo) GetAll() ([]manager.Project, error) {
	projects := []manager.Project{}
	repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(projectBucket)
		if b == nil {
			return nil
		}

		err := b.ForEach(func(key, value []byte) error {
			var project manager.Project
			buf := bytes.NewBuffer(value)
			err := gob.NewDecoder(buf).Decode(&project)
			if err != nil {
				return fmt.Errorf("unable to deserialize gob data: %v", err)
			}

			projects = append(projects, project)

			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to fetch bucket: %v", err)
		}

		return nil
	})

	return projects, nil
}

func (repo *projectRepo) GetByName(name string) (*manager.Project, error) {

	var project *manager.Project

	repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(projectBucket)
		if b == nil {
			return nil
		}

		value := b.Get([]byte(name))
		if value != nil {
			buf := bytes.NewBuffer(value)
			err := gob.NewDecoder(buf).Decode(&project)
			if err != nil {
				return fmt.Errorf("unable to deserialize gob data: %v", err)
			}
		}

		return nil
	})

	return project, nil
}

func (repo *projectRepo) Save(project manager.Project) error {
	// set created at if needed
	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
	}

	repo.db.Update(func(tx *bolt.Tx) error {
		// get or create the bucket
		b, err := tx.CreateBucketIfNotExists(projectBucket)
		if err != nil {
			return fmt.Errorf("unable to create bolt bucket: %v", err)
		}

		// serialize project
		buf := bytes.Buffer{}
		err = gob.NewEncoder(&buf).Encode(project)
		if err != nil {
			return fmt.Errorf("unable to serialize gob data: %v", err)
		}

		// put it into the bucket
		err = b.Put([]byte(project.Name), buf.Bytes())
		if err != nil {
			return fmt.Errorf("unable to put data in bucket: %v", err)
		}

		return nil
	})

	return nil
}
