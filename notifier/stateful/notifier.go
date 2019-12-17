package stateful

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/agence-webup/backr/manager"
	bolt "go.etcd.io/bbolt"
)

var notificationBucket = []byte("notifications")

// NewNotifier returns a notifier maintaining its state using bolt
func NewNotifier(db *bolt.DB) manager.Notifier {
	return &notifier{}
}

type notifier struct {
	db *bolt.DB
}

type notification struct {
	Statement manager.ProjectErrorStatement
	CreatedAt time.Time
	SentCount int
}

var delayBetweenSending = 6 * time.Hour

func (n *notifier) Notify(statement manager.ProjectErrorStatement) error {

	existingNotification, err := n.getNotificationForProject(statement.Project)
	if err != nil {
		return fmt.Errorf("unable to fetch an existing notification: %w", err)
	}

	if existingNotification != nil {
		// there is no issue remaining on this project, notify that everything is ok
		if statement.Count == 0 && existingNotification.Statement.Count > 0 {

		}

		trigger := existingNotification.CreatedAt.Add(delayBetweenSending * time.Duration(existingNotification.SentCount))
		if time.Now().Before(trigger) {
			// do nothing, the statement has already been notified
			return nil
		}
	}

	switch statement.MaxLevel {
	case manager.Warning:
		fmt.Println("*** ⚠️  WARNING ***")
	case manager.Critic:
		fmt.Println("*** 🆘  CRITICAL ***")
	}

	// fmt.Printf("→ %s\n", alert.Title)
	// fmt.Println(alert.Message)
	// j, _ := json.Marshal(alert.Metadata)
	// fmt.Println(string(j))
	// fmt.Println("——————————————————————")
	// fmt.Println("")

	return nil
}

func (n *notifier) getNotificationForProject(project manager.Project) (*notification, error) {
	var notif *notification

	err := n.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(notificationBucket)
		if b == nil {
			return nil
		}

		value := b.Get([]byte(project.Name))
		if value != nil {
			buf := bytes.NewBuffer(value)
			err := gob.NewDecoder(buf).Decode(&notif)
			if err != nil {
				return fmt.Errorf("unable to deserialize gob data: %v", err)
			}
		}

		return nil
	})

	return notif, err
}

func (n *notifier) save(notif notification) error {

	n.db.Update(func(tx *bolt.Tx) error {
		// get or create the bucket
		b, err := tx.CreateBucketIfNotExists(notificationBucket)
		if err != nil {
			return fmt.Errorf("unable to create bolt bucket: %w", err)
		}

		// serialize project
		buf := bytes.Buffer{}
		err = gob.NewEncoder(&buf).Encode(notif)
		if err != nil {
			return fmt.Errorf("unable to serialize gob data: %v", err)
		}

		// put it into the bucket
		err = b.Put([]byte(notif.Statement.Project.Name), buf.Bytes())
		if err != nil {
			return fmt.Errorf("unable to put data in bucket: %v", err)
		}

		return nil
	})

	return nil
}
