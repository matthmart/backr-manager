package process

import (
	"fmt"
	"testing"
	"time"

	"github.com/agence-webup/backr/manager/repositories/inmem"

	"github.com/agence-webup/backr/manager"
)

func TestProcessExecution(t *testing.T) {

	tests := getProcessTestCases()
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

			err := Execute(test.ReferenceDate, test.ProjectRepository, test.FileRepository)
			if err != nil {
				t.Fatalf("Execute returned an error: %v", err.Error())
			}

			Notify(test.ProjectRepository, test.Notifier)

			// check the state for each project in repo
			expectedProjectsStates, expectedFiles, expectedSentAlerts := test.Expected()
			projects, _ := test.ProjectRepository.GetAll()
			for _, project := range projects {
				actualState := project.State
				expectedState := expectedProjectsStates[project.Name]

				for ruleID, rs := range actualState {
					expected := expectedState[ruleID]

					t.Run(project.Name+"."+string(ruleID), func(t *testing.T) {
						// next date should be set
						t.Run("next date should be set", func(t *testing.T) {
							if rs.Next == nil {
								t.Errorf("next date is not set: expected=%v", expected.Next)
							}
						})

						// next date should be refDate + 24h when not already set (no initial state) OR should be the latest selected file date, added by (rule.MinAge * 24 hours)
						t.Run("next date should be refDate + 24h when not already set OR the latest selected file date, added by (rule.MinAge * 24 hours)", func(t *testing.T) {
							if rs.Next != nil && !expected.Next.Equal(*rs.Next) {
								t.Errorf("next date is wrong: expected=%v got=%v", expected.Next, rs.Next)
							}
						})

						// the files must be correctly selected
						t.Run("the files must be correctly selected", func(t *testing.T) {
							if len(rs.Files) != len(expected.Files) {
								t.Errorf("wrong number of selected files: expected=%+v got=%+v", len(expected.Files), len(rs.Files))
								return
							}

							for i := range rs.Files {
								rsFile := rs.Files[i]
								expectedFile := expected.Files[i]

								// check path
								if rsFile.Path != expectedFile.Path {
									t.Errorf("wrong file path: expected=%+v got=%+v", expectedFile.Path, rsFile.Path)
								}
								// check expiration date
								if !rsFile.Expiration.Equal(expectedFile.Expiration) {
									t.Errorf("wrong expiration date: expected:%+v got=%+v", expectedFile.Expiration, rsFile.Expiration)
								}
								// check file error
								if rsFile.Error != nil || expectedFile.Error != nil {
									rsFileError, ok1 := rsFile.Error.(*manager.RuleStateError)
									expectedFileError, ok2 := expectedFile.Error.(*manager.RuleStateError)
									if ok1 && ok2 {
										if rsFileError.Reason != expectedFileError.Reason {
											t.Errorf("wrong fileError reason: expected:%+v got=%+v", expectedFileError.Reason, rsFileError.Reason)
										}
										if rsFileError.File != expectedFileError.File {
											t.Errorf("wrong file associated to fileError: expected:%+v got=%+v", expectedFileError.File, rsFileError.File)
										}
									} else {
										t.Errorf("wrong fileError: expected:%+v got=%+v", expectedFile.Error, rsFile.Error)
									}
								}
							}
						})
					})
				}
			}

			t.Run("FileRepository must return the right files", func(t *testing.T) {
				// fetch all files from repo (after deletion during Execute)
				files, err := test.FileRepository.GetAll()
				if err != nil {
					t.Errorf("unable to get all files via repository: %v", err.Error())
				}

				// check if the files match with the expected ones
				filesPresent := map[string]int{}
				for _, expectedFile := range expectedFiles {
					filesPresent[expectedFile.Path] = 1
				}
				for _, file := range files {
					filesPresent[file.Path] = filesPresent[file.Path] + 1
				}
				for path, val := range filesPresent {
					if val == 1 {
						t.Errorf("file '%v' is missing from files", path)
					}
				}
			})

			t.Run("check sent notifications", func(t *testing.T) {
				test.Notifier.checkSentNotifications(t, expectedSentAlerts)
			})
		})
	}
}

type processTest struct {
	Name              string
	Description       string
	ReferenceDate     time.Time
	ProjectRepository manager.ProjectRepository
	FileRepository    manager.FileRepository
	Notifier          *testNotifier
	Expected          func() (map[string]manager.ProjectState, []manager.File, []manager.Alert)
}

func getProcessTestCases() []processTest {
	return []processTest{
		func() processTest {
			refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 20, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 23, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 3, 12, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 24, 3, 40, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file4.tar.gz", Date: time.Date(2019, 03, 24, 4, 23, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file5.tar.gz", Date: time.Date(2019, 03, 24, 5, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file6.tar.gz", Date: time.Date(2019, 03, 24, 6, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file7.tar.gz", Date: time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local), Size: 300},
			}

			initialState := manager.ProjectState{}
			initialNext := refDate.Add(-24 * time.Hour)
			initialState[rule.GetID()] = manager.RuleState{
				Rule: rule,
				Next: &initialNext,
				// Files: []manager.SelectedFile{},
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			return processTest{
				Name:              "everything's looking good",
				Description:       "no previous backup, no file problem",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := time.Date(2019, 03, 26, 5, 0, 0, 0, time.Local)
					expectedState[rule.GetID()] = manager.RuleState{
						Rule: rule,
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[6], Expiration: files[6].Date.Add(24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[7], files[6], files[1]}
					expectedSentAlerts := []manager.Alert{}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project2/file1.tar.gz", Date: time.Date(2019, 03, 23, 5, 0, 0, 0, time.Local), Size: 300},
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
				},
			}

			return processTest{
				Name:              "no file, no state",
				Description:       "No initial state, no available file for the project, so nothing must happen except that Next date must be set to refDate + 24h",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := time.Date(2019, 03, 26, 8, 0, 0, 0, time.Local)
					expectedState[rule.GetID()] = manager.RuleState{
						Rule:  rule,
						Next:  &expectedNext,
						Files: []manager.SelectedFile{
							// manager.SelectedFile{File: files[0], Expiration: files[0].Date.Add(24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[0]}
					expectedSentAlerts := []manager.Alert{}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
			rule1 := manager.Rule{Count: 3, MinAge: 1}
			rule2 := manager.Rule{Count: 2, MinAge: 15}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 20, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 23, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 3, 12, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 24, 3, 40, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file4.tar.gz", Date: time.Date(2019, 03, 24, 4, 23, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file5.tar.gz", Date: time.Date(2019, 03, 24, 5, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file6.tar.gz", Date: time.Date(2019, 03, 24, 6, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file7.tar.gz", Date: time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local), Size: 300},
			}

			initialState := manager.ProjectState{}
			initialNext := refDate.Add(-24 * time.Hour)
			initialState[rule1.GetID()] = manager.RuleState{
				Rule: rule1,
				Next: &initialNext,
			}
			initialState[rule2.GetID()] = manager.RuleState{
				Rule: rule2,
				Next: &initialNext,
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule1, rule2,
					},
					State: initialState,
				},
			}

			return processTest{
				Name:              "2 rules",
				Description:       "no previous backup, no file problem",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext1 := time.Date(2019, 03, 26, 5, 0, 0, 0, time.Local)
					expectedNext2 := time.Date(2019, 04, 9, 6, 0, 0, 0, time.Local) // daylight saving adds 1 hour
					expectedState[rule1.GetID()] = manager.RuleState{
						Rule: rule1,
						Next: &expectedNext1,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(time.Duration(rule1.MinAge) * 24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[6], Expiration: files[6].Date.Add(time.Duration(rule1.MinAge) * 24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(time.Duration(rule1.MinAge) * 24 * time.Hour), Error: nil},
						},
					}
					expectedState[rule2.GetID()] = manager.RuleState{
						Rule: rule2,
						Next: &expectedNext2,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(time.Duration(rule2.MinAge) * 24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[7], files[6], files[1]}
					expectedSentAlerts := []manager.Alert{}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 20, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 21, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 3, 12, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 24, 3, 40, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file4.tar.gz", Date: time.Date(2019, 03, 24, 4, 23, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file5.tar.gz", Date: time.Date(2019, 03, 24, 5, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file6.tar.gz", Date: time.Date(2019, 03, 24, 6, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file7.tar.gz", Date: time.Date(2019, 03, 25, 4, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file8.tar.gz", Date: time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local), Size: 5},
			}

			initialState := manager.ProjectState{}
			initialNext := refDate.Add(-24 * time.Hour)
			initialState[rule.GetID()] = manager.RuleState{
				Rule: rule,
				Next: &initialNext,
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			fileError := manager.RuleStateError{File: files[8], Reason: manager.RuleStateErrorSizeTooSmall}

			return processTest{
				Name:              "latest file has size issue",
				Description:       "no initial state, problem with latest backup",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := time.Date(2019, 03, 26, 4, 0, 0, 0, time.Local)
					expectedState[rule.GetID()] = manager.RuleState{
						Rule: rule,
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[8], Expiration: files[8].Date.Add(24 * time.Hour), Error: &fileError},
							manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[6], Expiration: files[6].Date.Add(24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{
								File:   files[1],
								Reason: manager.RuleStateErrorObsolete,
							}},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[8], files[7], files[6], files[1]}
					expectedSentAlerts := []manager.Alert{
						manager.Alert{
							Level: manager.Warning,
							Title: "Backup issue",
						},
					}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 26, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 20, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 21, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 3, 12, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 24, 3, 40, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file4.tar.gz", Date: time.Date(2019, 03, 24, 4, 23, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file5.tar.gz", Date: time.Date(2019, 03, 24, 5, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file6.tar.gz", Date: time.Date(2019, 03, 24, 6, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file7.tar.gz", Date: time.Date(2019, 03, 25, 4, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file8.tar.gz", Date: time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local), Size: 5},
				manager.File{Path: "project1/file9.tar.gz", Date: time.Date(2019, 03, 26, 5, 0, 0, 0, time.Local), Size: 310},
			}

			initialState := manager.ProjectState{}
			initialNext := time.Date(2019, 03, 26, 4, 0, 0, 0, time.Local)
			initialState[rule.GetID()] = manager.RuleState{
				Rule: rule,
				Next: &initialNext,
				Files: []manager.SelectedFile{
					manager.SelectedFile{File: files[8], Expiration: files[8].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{File: files[8], Reason: manager.RuleStateErrorSizeTooSmall}},
					manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(24 * time.Hour), Error: nil},
					manager.SelectedFile{File: files[6], Expiration: files[6].Date.Add(24 * time.Hour), Error: nil},
					manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{
						File:   files[1],
						Reason: manager.RuleStateErrorObsolete,
					}},
				},
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			file8Error := manager.RuleStateError{File: files[8], Reason: manager.RuleStateErrorSizeTooSmall}

			return processTest{
				Name:              "backup size issue is fixed",
				Description:       "previous file had a size issue, latest is correct",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := time.Date(2019, 03, 27, 5, 0, 0, 0, time.Local)
					expectedState[rule.GetID()] = manager.RuleState{
						Rule: rule,
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[8], Expiration: files[8].Date.Add(24 * time.Hour), Error: &file8Error},
							manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[6], Expiration: files[6].Date.Add(24 * time.Hour), Error: nil},
							manager.SelectedFile{File: files[9], Expiration: files[9].Date.Add(24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[9], files[7], files[6], files[8]}
					expectedSentAlerts := []manager.Alert{
						manager.Alert{
							Level: manager.Warning,
							Title: "Backup issue",
						},
					}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 04, 25, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 20, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 21, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 3, 12, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 24, 3, 40, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file4.tar.gz", Date: time.Date(2019, 03, 24, 4, 23, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file5.tar.gz", Date: time.Date(2019, 03, 24, 5, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file6.tar.gz", Date: time.Date(2019, 03, 24, 6, 34, 2, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file7.tar.gz", Date: time.Date(2019, 03, 24, 7, 34, 2, 0, time.Local), Size: 300},
			}

			initialState := manager.ProjectState{}
			initialNext := refDate.Add(-24 * time.Hour)
			initialState[rule.GetID()] = manager.RuleState{
				Rule: rule,
				Next: &initialNext,
				// Files: []manager.SelectedFile{},
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			generateObsoleteError := func(file manager.File) *manager.RuleStateError {
				return &manager.RuleStateError{
					File:   file,
					Reason: manager.RuleStateErrorObsolete,
				}
			}

			return processTest{
				Name:              "files are obsolete",
				Description:       "every file are obsolete, a critical alert must be thrown",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					// files are obsolete, so the Next date will not be updated by the date of an old file
					// so the current Next date remains the same
					expectedNext := time.Date(2019, 04, 24, 8, 0, 0, 0, time.Local)
					expectedState[rule.GetID()] = manager.RuleState{
						Rule: rule,
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[7], Expiration: files[7].Date.Add(24 * time.Hour), Error: generateObsoleteError(files[7])},
							manager.SelectedFile{File: files[6], Expiration: files[6].Date.Add(24 * time.Hour), Error: generateObsoleteError(files[6])},
							manager.SelectedFile{File: files[5], Expiration: files[5].Date.Add(24 * time.Hour), Error: generateObsoleteError(files[5])},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[7], files[6], files[5]}
					expectedSentAlerts := []manager.Alert{
						manager.Alert{
							Level: manager.Critic,
							Title: "Backup issue",
						},
					}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 30, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 22, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file1.tar.gz", Date: time.Date(2019, 03, 23, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file2.tar.gz", Date: time.Date(2019, 03, 24, 5, 0, 0, 0, time.Local), Size: 300},
				manager.File{Path: "project1/file3.tar.gz", Date: time.Date(2019, 03, 30, 5, 0, 0, 0, time.Local), Size: 300},
			}

			initialState := manager.ProjectState{}
			initialNext := time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local)
			initialState[rule.GetID()] = manager.RuleState{
				Rule: rule,
				Next: &initialNext,
				Files: []manager.SelectedFile{
					manager.SelectedFile{File: files[2], Expiration: files[2].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{File: files[2], Reason: manager.RuleStateErrorObsolete}},
					manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{File: files[1], Reason: manager.RuleStateErrorObsolete}},
					manager.SelectedFile{File: files[0], Expiration: files[0].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{File: files[0], Reason: manager.RuleStateErrorObsolete}},
				},
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			return processTest{
				Name:              "kept files are obsolete, but it's fixed",
				Description:       "initial state containing only obsolete files, but a new fresh backup is coming",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := time.Date(2019, 03, 31, 6, 0, 0, 0, time.Local) // daylight saving adds 1 hour
					expectedState[rule.GetID()] = manager.RuleState{
						Rule: rule,
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[2], Expiration: files[2].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{File: files[2], Reason: manager.RuleStateErrorObsolete}},
							manager.SelectedFile{File: files[1], Expiration: files[1].Date.Add(24 * time.Hour), Error: &manager.RuleStateError{File: files[1], Reason: manager.RuleStateErrorObsolete}},
							manager.SelectedFile{File: files[3], Expiration: files[3].Date.Add(24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[3], files[2], files[1]}
					expectedSentAlerts := []manager.Alert{
						manager.Alert{
							Level: manager.Warning,
							Title: "Backup issue",
						},
					}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{}

			initialState := manager.ProjectState{}
			initialNext := refDate.Add(time.Duration(-24) * time.Hour)
			initialState[rule.GetID()] = manager.RuleState{
				Rule: rule,
				Next: &initialNext,
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			return processTest{
				Name:              "no file, but with initial state",
				Description:       "The next date has been set by a previous Execute run, so we expect to have some backup files. But there is any, so a critical error must be thrown.",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := initialNext
					expectedState[rule.GetID()] = manager.RuleState{
						Rule:  rule,
						Next:  &expectedNext,
						Files: []manager.SelectedFile{
							// manager.SelectedFile{File: files[0], Expiration: files[0].Date.Add(24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{}
					expectedSentAlerts := []manager.Alert{
						manager.Alert{
							Level: manager.Critic,
							Title: "Backup issue",
						},
					}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
		func() processTest {
			refDate := time.Date(2019, 03, 25, 8, 0, 0, 0, time.Local)
			rule := manager.Rule{Count: 3, MinAge: 1}
			files := []manager.File{
				manager.File{Path: "project1/file0.tar.gz", Date: time.Date(2019, 03, 25, 5, 0, 0, 0, time.Local), Size: 300},
			}

			initialState := manager.ProjectState{}
			initialNext := refDate.Add(time.Duration(-24) * time.Hour)
			initialState[rule.GetID()] = manager.RuleState{
				Rule:  rule,
				Next:  &initialNext,
				Error: &manager.RuleStateError{Reason: manager.RuleStateErrorNoFile},
			}

			projects := []manager.Project{
				manager.Project{
					Name: "project1",
					Rules: []manager.Rule{
						rule,
					},
					State: initialState,
				},
			}

			return processTest{
				Name:              "no file previously, but it's fixed",
				Description:       "There was no file, so an alert was thrown. A new file has been uploaded, so the error must be fixed.",
				ReferenceDate:     refDate,
				ProjectRepository: newMockProjectRepository(projects),
				FileRepository:    newMockFileRepository(files),
				Notifier:          newTestNotifier(),
				Expected: func() (map[string]manager.ProjectState, []manager.File, []manager.Alert) {
					expectedState := manager.ProjectState{}
					expectedNext := time.Date(2019, 03, 26, 5, 0, 0, 0, time.Local)
					expectedState[rule.GetID()] = manager.RuleState{
						Rule: rule,
						Next: &expectedNext,
						Files: []manager.SelectedFile{
							manager.SelectedFile{File: files[0], Expiration: files[0].Date.Add(24 * time.Hour), Error: nil},
						},
					}

					statesByProjectName := map[string]manager.ProjectState{
						"project1": expectedState,
					}
					expectedFilesInRepo := []manager.File{files[0]}
					expectedSentAlerts := []manager.Alert{}

					return statesByProjectName, expectedFilesInRepo, expectedSentAlerts
				},
			}
		}(),
	}
}

func printFiles(files []manager.SelectedFile) {
	fmt.Println("")
	for _, f := range files {
		fmt.Printf("  %+v\n", f)
	}
	fmt.Println("")
}

func newMockProjectRepository(projects []manager.Project) manager.ProjectRepository {
	r := inmem.NewProjectRepository()
	for _, p := range projects {
		r.Save(p)
	}

	return r
}

func newMockFileRepository(files []manager.File) manager.FileRepository {
	r := inmem.NewFileRepository()
	for _, f := range files {
		inmem.CreateFakeFile(r, f)
	}

	return r
}

func newTestNotifier() *testNotifier {
	n := testNotifier{
		sentNotifications: []manager.Alert{},
	}
	return &n
}

type testNotifier struct {
	sentNotifications []manager.Alert
}

func (not *testNotifier) Send(alert manager.Alert) {
	not.sentNotifications = append(not.sentNotifications, alert)
}

func (not *testNotifier) checkSentNotifications(t *testing.T, expectedSentNotifications []manager.Alert) {

	if len(not.sentNotifications) != len(expectedSentNotifications) {
		t.Fatalf("unexpected alerts count: expected=%d got=%d", len(expectedSentNotifications), len(not.sentNotifications))
	}

	for i, alert := range not.sentNotifications {
		expectedAlert := expectedSentNotifications[i]

		if alert.Level != expectedAlert.Level {
			t.Errorf("unexpected alert level: expected=%v got=%v", expectedAlert.Level, alert.Level)
		}
		if alert.Title != expectedAlert.Title {
			t.Errorf("unexpected alert title: expected=%v got=%v", expectedAlert.Title, alert.Title)
		}
	}
}
