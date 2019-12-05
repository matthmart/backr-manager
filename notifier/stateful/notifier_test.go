package stateful

import (
	"testing"
	"time"

	"github.com/agence-webup/backr/manager"
	bolt "go.etcd.io/bbolt"
)

// func TestMain(m *testing.M) {

// 	testDB = db

// 	// execute tests
// 	returnCode := m.Run()

// 	// cleanup DB file
// 	testDB.Close()
// 	// os.Remove("notifier_test.db")

// 	os.Exit(returnCode)
// }

type testContext struct {
	DB *bolt.DB
}

func TestShouldNotifyWhenThereIsNoIssueAnymore(t *testing.T) {
	ctx := setupTest()

	n := notifier{db: ctx.DB}

	fakeStatement := manager.ProjectErrorStatement{
		Project:  manager.Project{Name: "test"},
		MaxLevel: manager.Warning,
		Count:    1,
	}
	fakeExistingNotif := notification{
		Statement: fakeStatement,
		SentCount: 1,
	}
	n.save(fakeExistingNotif)

	// teardownTest(ctx)
}

func setupTest() testContext {
	// create a test DB file
	db, err := bolt.Open("notifier_test.db", 0666, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(err)
	}

	return testContext{
		DB: db,
	}
}

func teardownTest(ctx testContext) {
	ctx.DB.Close()
}
