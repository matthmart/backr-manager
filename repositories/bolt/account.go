package bolt

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/agence-webup/backr/manager"
	"github.com/agence-webup/backr/manager/bcrypt"
	bolt "go.etcd.io/bbolt"
)

func NewAccountRepository(db *bolt.DB) manager.AccountRepository {
	r := accountRepository{
		db: db,
	}

	return &r
}

var accountBucket = []byte("accounts")

type accountRepository struct {
	db *bolt.DB
}

func (repo *accountRepository) List() ([]manager.Account, error) {
	accounts := []manager.Account{}
	repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountBucket)
		if b == nil {
			return nil
		}

		err := b.ForEach(func(key, value []byte) error {
			var account manager.Account
			buf := bytes.NewBuffer(value)
			err := gob.NewDecoder(buf).Decode(&account)
			if err != nil {
				return fmt.Errorf("unable to deserialize gob data: %v", err)
			}

			accounts = append(accounts, account)

			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to fetch bucket: %v", err)
		}

		return nil
	})

	return accounts, nil
}

func (repo *accountRepository) Get(username string) (*manager.Account, error) {
	var account *manager.Account
	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountBucket)
		if b == nil {
			return nil
		}

		value := b.Get([]byte(username))

		var acc manager.Account
		buf := bytes.NewBuffer(value)
		err := gob.NewDecoder(buf).Decode(&acc)
		if err != nil {
			return fmt.Errorf("unable to deserialize gob data: %v", err)
		}

		account = &acc

		return nil
	})

	return account, err
}

func (repo *accountRepository) Create(username string) (string, error) {

	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}

	pwd, err := bcrypt.GeneratePassword()
	if err != nil {
		return "", fmt.Errorf("unable to generate password: %v", err)
	}

	account := manager.Account{
		Username:       username,
		HashedPassword: pwd.Hashed,
	}

	repo.db.Update(func(tx *bolt.Tx) error {
		// get or create the bucket
		b, err := tx.CreateBucketIfNotExists(accountBucket)
		if err != nil {
			return fmt.Errorf("unable to create bolt bucket: %v", err)
		}

		// serialize account
		buf := bytes.Buffer{}
		err = gob.NewEncoder(&buf).Encode(account)
		if err != nil {
			return fmt.Errorf("unable to serialize gob data: %v", err)
		}

		// put it into the bucket
		err = b.Put([]byte(username), buf.Bytes())
		if err != nil {
			return fmt.Errorf("unable to put data in bucket: %v", err)
		}

		return nil
	})

	return pwd.Plain, nil
}

func (repo *accountRepository) Delete(username string) error {

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	return repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountBucket)
		if b == nil {
			return nil
		}

		err := b.Delete([]byte(username))
		if err != nil {
			return fmt.Errorf("unable to delete bolt key: %v", err)
		}
		return nil
	})
}

func (repo *accountRepository) Authenticate(username string, password string) error {

	return repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountBucket)
		if b == nil {
			return fmt.Errorf("wrong credentials: bucket does not exist")
		}

		value := b.Get([]byte(username))
		if value == nil {
			return fmt.Errorf("wrong credentials: username does not exist")
		}

		// unserialize account
		var account manager.Account
		buf := bytes.NewBuffer(value)
		err := gob.NewDecoder(buf).Decode(&account)
		if err != nil {
			return fmt.Errorf("wrong credentials: %v", err)
		}

		err = bcrypt.CompareHashAndPassword(account.HashedPassword, password)
		if err != nil {
			return fmt.Errorf("wrong credentials: %v", err)
		}

		return nil
	})
}

func (repo *accountRepository) ChangePassword(username string) (string, error) {

	account, err := repo.Get(username)
	if err != nil {
		return "", fmt.Errorf("unable to get account: %v", err)
	}
	if account == nil {
		return "", fmt.Errorf("account not found")
	}

	pwd, err := bcrypt.GeneratePassword()
	if err != nil {
		return "", fmt.Errorf("unable to generate password: %v", err)
	}

	account.HashedPassword = pwd.Hashed

	err = repo.db.Update(func(tx *bolt.Tx) error {
		// get or create the bucket
		b := tx.Bucket(accountBucket)
		if b == nil {
			return fmt.Errorf("unable to get bucket")
		}

		// serialize project
		buf := bytes.Buffer{}
		err = gob.NewEncoder(&buf).Encode(account)
		if err != nil {
			return fmt.Errorf("unable to serialize gob data: %v", err)
		}

		// put it into the bucket
		err = b.Put([]byte(username), buf.Bytes())
		if err != nil {
			return fmt.Errorf("unable to put data in bucket: %v", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return pwd.Plain, nil
}
