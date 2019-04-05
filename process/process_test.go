package process

import (
	"fmt"
	"testing"
	"time"

	"github.com/agence-webup/backr/manager/notifier/basic"
	"github.com/agence-webup/backr/manager/repositories/inmem"

	"github.com/agence-webup/backr/manager"
)

func TestProcessManagerSelectFilesForRuleState(t *testing.T) {
	refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
	notifier := basic.NewNotifier()

	tests := getTestCases()

	pm := processManager{
		fileRepo:      inmem.NewFileRepository(),
		projectRepo:   inmem.NewProjectRepository(),
		notifier:      notifier,
		referenceDate: refDate,
	}
	// files := []manager.File{
	// 	manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 23, 5, 0, 0, 0, time.Local), Size: 300},
	// 	// manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 5, 0, 0, 0, time.Local), Size: 300},
	// 	// manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local), Size: 300},
	// }
	errCollector := newErrorCollector(notifier)

	// pm.selectFilesToBackup(&rs, files, errCollector)

	// // fmt.Printf("rs: %+v\n", rs)

	// // t.Fatal()

	// if !expectedState.Next.Equal(*rs.Next) {
	// 	t.Fatalf("next date is wrong: expected=%v got=%v", expectedState.Next, rs.Next)
	// }

	for _, test := range tests {
		err := runRuleStateTest(test, pm, errCollector)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// func TestProcessManagerSelectFilesShouldTakeMostRecent(t *testing.T) {
// 	tests := getTestCases()

// 	for _, test := range tests {
// 		rs := test.Processed()

// 		latestFile := test.Files[len(test.Files)-1]

// 	}
// }

func getTestCases() []ruleStateTest {
	// refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
	// notifier := basic.NewNotifier()
	// errCollector := newErrorCollector(notifier)
	// pm := processManager{
	// 	fileRepo:      inmem.NewFileRepository(),
	// 	projectRepo:   inmem.NewProjectRepository(),
	// 	notifier:      notifier,
	// 	referenceDate: refDate,
	// }

	return []ruleStateTest{
		func() ruleStateTest {
			files := []manager.File{
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 23, 5, 0, 0, 0, time.Local), Size: 300},
			}
			return ruleStateTest{
				Files: files,
				Initial: func() manager.RuleState {
					rs := manager.RuleState{
						Rule:  manager.Rule{Count: 3, MinAge: 1},
						Next:  nil,
						Files: []manager.SelectedFile{},
					}
					// pm.selectFilesToBackup(&rs, files, errCollector)
					return rs
				},
				Expected: func() manager.RuleState {
					expectedNext := time.Date(2019, 03, 24, 5, 0, 0, 0, time.Local)
					return manager.RuleState{
						Rule: manager.Rule{Count: 3, MinAge: 1},
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[0], Expiration: files[0].Date.Add(24 * time.Hour), Error: nil},
						},
					}
				},
			}
		}(),
		func() ruleStateTest {
			files := []manager.File{
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 20, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 3, 12, 2, 0, time.Local), Size: 300},
			}
			return ruleStateTest{
				Files: files,
				Initial: func() manager.RuleState {
					rs := manager.RuleState{
						Rule:  manager.Rule{Count: 3, MinAge: 1},
						Next:  nil,
						Files: []manager.SelectedFile{},
					}
					return rs
				},
				Expected: func() manager.RuleState {
					expectedNext := time.Date(2019, 03, 25, 3, 12, 2, 0, time.Local)
					return manager.RuleState{
						Rule: manager.Rule{Count: 3, MinAge: 1},
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(24 * time.Hour), Error: nil},
						},
					}
				},
			}
		}(),
	}
}

type ruleStateTest struct {
	Files    []manager.File
	Initial  func() manager.RuleState
	Expected func() manager.RuleState
}

func runRuleStateTest(test ruleStateTest, pm processManager, errCollector errorCollector) error {
	rs := test.Initial()
	expected := test.Expected()

	pm.selectFilesToBackup(&rs, test.Files, errCollector)

	// next date should be set
	if rs.Next == nil {
		return fmt.Errorf("next date is not set: expected=%v", expected.Next)
	}

	// next date should be the latest selected file date, added by (rule.MinAge * 24 hours)
	if !expected.Next.Equal(*rs.Next) {
		return fmt.Errorf("next date is wrong: expected=%v got=%v", expected.Next, rs.Next)
	}

	return nil
}
