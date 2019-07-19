package manager

import (
	"testing"
	"time"
)

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

	future := time.Date(2019, 03, 28, 8, 0, 0, 0, time.UTC)
	rs = getSampleRuleState(&future)
	mustBackup = rs.Check(base)
	if mustBackup == true {
		t.Fatal("RuleState with a future next date should not trigger a backup")
	}

}

func TestFilesByDateDesc(t *testing.T) {
	sampleFiles := getSampleFiles()
	expected := []string{"file2", "file1", "file3"}

	result := FilesSortedByDateDesc(sampleFiles)

	for i := 0; i < len(sampleFiles); i++ {
		if expected[i] != result[i].Path {
			t.Fatalf("wrong order: got=%v expected=%v", result, expected)
		}
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

func getSampleFiles() []File {
	return []File{
		File{Path: "file1", Date: time.Date(2019, 03, 26, 8, 0, 0, 0, time.UTC)},
		File{Path: "file2", Date: time.Date(2019, 03, 26, 10, 0, 0, 0, time.UTC)},
		File{Path: "file3", Date: time.Date(2019, 03, 26, 6, 0, 0, 0, time.UTC)},
	}
}
