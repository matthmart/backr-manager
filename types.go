package manager

import (
	"fmt"
	"sort"
	"time"
)

// ProjectState represents the state for each rule associated to the project
type ProjectState map[RuleID]RuleState

// Project represents a configured project in the manager
type Project struct {
	Name      string
	Rules     []Rule
	State     ProjectState
	CreatedAt time.Time
}

// UpdateState update the rule state of the project, for the specified the ruleID, using the state passed as parameter
func (project *Project) UpdateState(ruleID RuleID, state RuleState) error {
	if project.State == nil {
		project.State = ProjectState{}
	}

	project.State[ruleID] = state

	return nil
}

// RemoveFilesFromState takes a list of files that are just being removed and
// and update the project state accordingly, for each rule
func (project *Project) RemoveFilesFromState(removedFiles []File) {

	for _, rs := range project.State {
		files := []SelectedFile{}
		for _, f := range rs.Files {
			mustBeRemoved := false
			for _, removedFile := range removedFiles {
				if f.Path == removedFile.Path {
					mustBeRemoved = true
					break
				}
			}

			if !mustBeRemoved {
				files = append(files, f)
			}
		}
		rs.Files = files
		project.UpdateState(rs.Rule.GetID(), rs)
	}

}

// DebugPrint outputs debug information on project
func (project *Project) DebugPrint() {
	fmt.Printf("name: %v\n", project.Name)
	for id, rs := range project.State {
		fmt.Printf(" - %v\n", id)
		for _, f := range rs.Files {
			errState := "✅"
			if f.Error != nil {
				switch f.Error.Reason {
				case RuleStateErrorSizeTooSmall:
					errState = "⚠️ too small"
				case RuleStateErrorObsolete:
					errState = "🆘 obsolete"
				}
			}
			fmt.Printf("   %v [%v] %v\n", f.Path, f.Expiration, errState)
		}
	}
	fmt.Println("")
}

// Rule defines the spec of a backup lifetime management
type Rule struct {
	Count  int
	MinAge int
}

// GetID returns the ID identifying the rule (in project rules scope)
func (r Rule) GetID() RuleID {
	return RuleID(fmt.Sprintf("rule%d.%d", r.Count, r.MinAge))
}

// RuleID represents an unique identifier for a rule
type RuleID string

// RuleState stores the current state for a rule:
// i.e. next backup date, selected backup files
type RuleState struct {
	Rule  Rule
	Files []SelectedFile
	Next  *time.Time

	// global error (when the error is not associated to a specific file)
	Error *RuleStateError
}

// Check takes a date (e.g. today) and checks if the backup must be done
// according to the `Next` field.
func (rs *RuleState) Check(baseDate time.Time) bool {
	// when Next is not set, it's because the process has not been executed yet (files might not be available)
	// so consider it's not necessary to perform a backup now, waiting for the Next date to be set
	if rs.Next == nil {
		return false
	}

	// the process must not be executed when the Next date is not reached yet
	if (*rs.Next).After(baseDate) {
		return false
	}

	return true
}

// RuleStateErrorType represents the reason of a RuleStateError
type RuleStateErrorType int

const (
	// RuleStateErrorObsolete indicates that a file is expired
	RuleStateErrorObsolete RuleStateErrorType = iota
	// RuleStateErrorSizeTooSmall indicates that a file size seems too small
	RuleStateErrorSizeTooSmall
	// RuleStateErrorNoFile indicates that backup files are missing (no specific file is linked)
	RuleStateErrorNoFile
)

func (r RuleStateErrorType) String() string {
	reason := ""
	switch r {
	case RuleStateErrorObsolete:
		reason = "outdated"
	case RuleStateErrorSizeTooSmall:
		reason = "file is too small"
	case RuleStateErrorNoFile:
		reason = "no available file"
	default:
		reason = "unknown error"
	}
	return reason
}

// RuleStateError represents an error on a rule
type RuleStateError struct {
	File   File
	Reason RuleStateErrorType
}

func (e *RuleStateError) Error() string {
	f := File{}
	if e.File == f {
		return e.Reason.String()
	}

	return "error for '" + e.File.Path + "': " + e.Reason.String()
}

// RulesByMinAge stores a slice of rules, sorted by min age
type RulesByMinAge []Rule

func (r RulesByMinAge) Len() int {
	return len(r)
}

func (r RulesByMinAge) Less(i, j int) bool {
	return r[i].MinAge < r[j].MinAge
}

func (r RulesByMinAge) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// File represents a backup file
type File struct {
	Path string
	Date time.Time
	Size int64
}

// FilesByFolder represents files mapped by their parent folder
type FilesByFolder map[string][]File

// FilesSortedByDateDesc returns a slice of files,
// sorted by date from earlier to older
func FilesSortedByDateDesc(files []File) []File {
	f := make([]File, len(files))
	copy(f, files)

	sorted := sortedFilesByDate(f)
	sort.Sort(sort.Reverse(sorted))

	return sorted
}

type sortedFilesByDate []File

func (files sortedFilesByDate) Len() int {
	return len(files)
}
func (files sortedFilesByDate) Less(i, j int) bool {
	return files[i].Date.Before(files[j].Date)
}
func (files sortedFilesByDate) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

// SelectedFile represents a file that is selected for a rule
type SelectedFile struct {
	File
	Expiration time.Time
	Error      *RuleStateError
}

// SelectedFilesByExpirationDateDesc stores a slice of files (associated to a rule state),
// which should be sorted by expiration date, from older to earlier
type SelectedFilesByExpirationDateDesc []SelectedFile

// SelectedFilesSortedByExpirationDateDesc takes a list of SelectedFile and sort them by expiration date in desc order
func SelectedFilesSortedByExpirationDateDesc(files []SelectedFile) SelectedFilesByExpirationDateDesc {
	f := make([]SelectedFile, len(files))
	copy(f, files)

	sorted := selectedFilesByExpirationDate(f)
	sort.Sort(sort.Reverse(sorted))

	return SelectedFilesByExpirationDateDesc(sorted)
}

type selectedFilesByExpirationDate []SelectedFile

func (files selectedFilesByExpirationDate) Len() int {
	return len(files)
}
func (files selectedFilesByExpirationDate) Less(i, j int) bool {
	return files[i].Expiration.Before(files[j].Expiration)
}
func (files selectedFilesByExpirationDate) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}
