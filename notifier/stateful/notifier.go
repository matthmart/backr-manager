package stateful

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/agence-webup/backr/manager"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

var notificationBucket = []byte("notifications")

// NewNotifier returns a notifier maintaining its state using bolt
func NewNotifier(db *bolt.DB, config manager.SlackNotifierConfig) manager.Notifier {
	return &notifier{
		db:         db,
		webhookURL: config.WebhookURL,
	}
}

type notifier struct {
	db         *bolt.DB
	webhookURL string
}

type notification struct {
	Statement manager.ProjectErrorStatement
	CreatedAt time.Time
	SentAt    time.Time
}

// var delayBetweenSending = 6 * time.Hour
var delayBetweenSending = 10 * time.Minute

func (n *notifier) Notify(statement manager.ProjectErrorStatement) error {

	existingNotification, err := n.getNotificationForStatement(statement)
	if err != nil {
		return fmt.Errorf("unable to fetch an existing notification: %w", err)
	}

	if existingNotification != nil {
		log.Debug().Caller().Str("project_name", existingNotification.Statement.Project.Name).Msg("found existing notification for statement")

		// // there is no issue remaining on this project, notify that everything is ok
		// if statement.Count == 0 && existingNotification.Statement.Count > 0 {
		// 	existingNotification.Statement = statement
		// 	log.Info().Str("project_name", existingNotification.Statement.Project.Name).Msg("notify: issue is resolved")
		// 	sendSlackMessage(n.webhookURL, existingNotification)

		// 	return nil
		// }

		trigger := existingNotification.SentAt.Add(delayBetweenSending)
		fmt.Println(trigger)
		if time.Now().Before(trigger) {
			// do nothing, the statement has already been notified
			log.Info().Str("project_name", existingNotification.Statement.Project.Name).Str("created_at", existingNotification.CreatedAt.String()).Str("sent_at", existingNotification.SentAt.String()).Msg("notify: issue was already notified")
			return nil
		}
	}

	// create notification
	var notif notification
	if existingNotification != nil {
		// update existing notification
		notif = *existingNotification
	} else {
		// create new notification
		notif = notification{Statement: statement, CreatedAt: time.Now()}
	}
	notif.SentAt = time.Now()

	// notify for issue
	sendSlackMessage(n.webhookURL, notif)

	// save notification
	n.save(notif)

	log.Info().Str("project_name", notif.Statement.Project.Name).Msg("notify: backup issue")

	return nil
}

func (n *notifier) getNotificationForStatement(statement manager.ProjectErrorStatement) (*notification, error) {
	var notif *notification

	err := n.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(notificationBucket)
		if b == nil {
			return nil
		}

		value := b.Get([]byte(statement.GetUniqueID()))
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

		// serialize notification
		buf := bytes.Buffer{}
		err = gob.NewEncoder(&buf).Encode(notif)
		if err != nil {
			return fmt.Errorf("unable to serialize gob data: %v", err)
		}

		// put it into the bucket
		err = b.Put([]byte(notif.Statement.GetUniqueID()), buf.Bytes())
		if err != nil {
			return fmt.Errorf("unable to put data in bucket: %v", err)
		}

		return nil
	})

	return nil
}
