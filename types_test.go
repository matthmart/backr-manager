package manager

import (
	"testing"
	"time"
)

// func TestS3FilesByDateAscLatest(t *testing.T) {

// 	files := getSampleFiles()

// 	f := files.Latest()

// 	expected := "file2"
// 	if f.Filename != expected {
// 		t.Errorf("sorting failed: expected: %s got: %s", expected, f.Filename)
// 	}
// }

func TestRuleStateCheck(t *testing.T) {
	base := time.Date(2019, 03, 26, 8, 0, 0, 0, time.UTC)

	rs := getSampleRuleState(nil)
	mustBackup := rs.Check(base)
	if mustBackup == true {
		t.Fatal("RuleState with no next date should not trigger a backup, it must be set before")
	}

	old := time.Date(2019, 03, 24, 8, 0, 0, 0, time.UTC)
	rs = getSampleRuleState(&old)
	mustBackup = rs.Check(base)
	if mustBackup == false {
		t.Fatal("RuleState with an old next date should trigger a backup")
	}

}

func getSampleRuleState(next *time.Time) RuleState {
	return RuleState{
		Rule: Rule{
			Count:  3,
			MinAge: 1,
		},
		Next: next,
	}
}
