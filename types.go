package manager

import (
	"fmt"
	"sort"
	"time"
)

// Project represents a configured project in the manager
type Project struct {
	Name  string
	Rules []Rule
	State map[RuleID]RuleState
}

func (project *Project) UpdateState(ruleID RuleID, state RuleState) error {
	if project.State == nil {
		project.State = map[RuleID]RuleState{}
	}

	project.State[ruleID] = state

	return nil
}

// func (project *Project) GetFilesToRemove(allFiles []File) []File {

// 	filesToKeep := map[string]int{}
// 	for _, rs := range project.State {
// 		for _, f := range rs.Files {
// 			filesToKeep[f.Path] = 1
// 		}
// 	}

// 	// fmt.Printf("[%v] files to keep: %v\n", project.Name, project.State)

// 	filesToRemove := []File{}
// 	for _, f := range allFiles {
// 		if _, ok := filesToKeep[f.Path]; !ok {
// 			filesToRemove = append(filesToRemove, f)
// 		}
// 	}

// 	return filesToRemove

// 	// filesToKeep := mapset.NewSet()
// 	// for _, rs := range project.State {
// 	// 	for _, b := range rs.Files {
// 	// 		filesToKeep.Add(b)
// 	// 	}
// 	// }

// 	// files := mapset.NewSet()
// 	// for _, b := range allFiles {
// 	// 	files.Add(b)
// 	// }

// 	// filesToRemove := []S3File{}
// 	// for _, f := range files.Difference(filesToKeep).ToSlice() {
// 	// 	filesToRemove = append(filesToRemove, f.(S3File))
// 	// }

// 	// return filesToRemove
// }

func (project *Project) GetFilesToRemove(allFiles []File, referenceDate time.Time) []File {

	filesMaxExpiration := map[string]time.Time{}
	for _, rs := range project.State {

		for _, f := range rs.Files {
			if expiration, ok := filesMaxExpiration[f.Path]; ok {
				if f.Expiration.Before(expiration) {
					continue
				}
			}

			filesMaxExpiration[f.Path] = f.Expiration
		}
	}

	filesToKeep := map[string]bool{}
	for _, rs := range project.State {

		filesByExpDateDesc := SelectedFilesByExpirationAsc(rs.Files)
		filesByExpDateDesc.Reverse()

		fileKeptCount := 0
		for _, f := range filesByExpDateDesc {

			maxExpiration := filesMaxExpiration[f.Path]
			// if maxExpiration.Before(now) && fileKeptCount > rs.Rule.Count {
			// 	filesToKeep[f.Path] = false
			// } else {
			// 	filesToKeep[f.Path] = true
			// 	fileKeptCount++
			// }
			fmt.Printf("%v: (fileKeptCount(%v) < rs.Rule.Count(%v) || maxExp(%v).After(%v))  && CanKeepFileForError(%v)\n", f.Path, fileKeptCount, rs.Rule.Count, maxExpiration, referenceDate, CanKeepFileForError(f.Error))
			if (fileKeptCount < rs.Rule.Count || maxExpiration.After(referenceDate)) && CanKeepFileForError(f.Error) {
				filesToKeep[f.Path] = true

				// fmt.Printf("  canKeepFile(%v)\n", CanKeepFileForError(f.Error))
				if CanKeepFileForError(f.Error) {
					fileKeptCount++
				}
			}
		}
		// }
	}

	fmt.Printf("[%v] files to keep: %+v\n", project.Name, filesToKeep)

	filesToRemove := []File{}
	for _, f := range allFiles {
		if _, ok := filesToKeep[f.Path]; !ok {
			filesToRemove = append(filesToRemove, f)
		}
	}

	return filesToRemove

	// filesToKeep := mapset.NewSet()
	// for _, rs := range project.State {
	// 	for _, b := range rs.Files {
	// 		filesToKeep.Add(b)
	// 	}
	// }

	// files := mapset.NewSet()
	// for _, b := range allFiles {
	// 	files.Add(b)
	// }

	// filesToRemove := []S3File{}
	// for _, f := range files.Difference(filesToKeep).ToSlice() {
	// 	filesToRemove = append(filesToRemove, f.(S3File))
	// }

	// return filesToRemove
}

func (project *Project) CheckForIssues() []RuleStateError {

	errors := []RuleStateError{}
	now := time.Now()

	for _, rs := range project.State {
		// check for file expiration
		for _, f := range rs.Files {
			if f.Expiration.Before(now) {
				errors = append(errors, RuleStateError{
					RuleState: rs,
					File:      f.File,
					Reason:    RuleStateErrorObsolete,
				})
			}
		}
	}

	return errors
}

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

func (p *Project) DebugPrint() {
	fmt.Printf("name: %v\n", p.Name)
	for id, rs := range p.State {
		fmt.Printf(" - %v\n", id)
		for _, f := range rs.Files {
			fmt.Printf("   %v [%v]\n", f.Path, f.Expiration)
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
	Rule             Rule
	Files            []SelectedFile
	Next             *time.Time
	PreviousFileSize *int64
}

// Check takes a date (e.g. today) and checks if the backup must be done
// according to the `Next` field.
func (rs *RuleState) Check(baseDate time.Time) bool {
	if rs.Next == nil {
		return false
	}
	if (*rs.Next).After(baseDate) {
		return false
	}
	return true
}

// Keep appends the file to the state (the one that must be kept)
// and set the next backup time
func (rs *RuleState) Keep(file File, someError error) error {
	// prepare a map with existing files
	filesByName := map[string]SelectedFile{}
	for _, f := range rs.Files {
		filesByName[f.Path] = f
	}

	// check if the file is not already kept
	if existingFile, ok := filesByName[file.Path]; !ok {
		f := SelectedFile{
			File:       file,
			Expiration: file.Date.Add(time.Duration(rs.Rule.MinAge) * 24 * time.Hour),
			Error:      someError,
		}
		rs.Files = append(rs.Files, f)
	} else {
		// update the eventual error
		existingFile.Error = someError
		filesByName[file.Path] = existingFile
	}

	// update state
	unit := 24 * time.Hour
	// unit := 1 * time.Minute
	next := file.Date.Add(time.Duration(rs.Rule.MinAge) * unit)
	if rs.Next == nil || next.After(*rs.Next) {
		rs.Next = &next
	}
	// rs.PreviousFileSize = &file.Size

	return nil
}

// // Clear removes the files that are not needed anymore
// func (rs *RuleState) Clear() {
// 	files := SelectedFilesByDateAsc(rs.Files)
// 	// ensure that files are sorted
// 	files.Sort()
// 	// remove the oldest files
// 	if len(files) > rs.Rule.Count {
// 		from := len(files) - rs.Rule.Count
// 		rs.Files = files[from:]
// 	}
// }

type RuleStateErrorType int

const (
	RuleStateErrorObsolete RuleStateErrorType = iota
	RuleStateErrorSizeTooSmall
)

func (r RuleStateErrorType) String() string {
	reason := ""
	switch r {
	case RuleStateErrorObsolete:
		reason = "outdated"
	case RuleStateErrorSizeTooSmall:
		reason = "file is too small"
	default:
		reason = "unknown error"
	}
	return reason
}

type RuleStateError struct {
	RuleState RuleState
	File      File
	Reason    RuleStateErrorType
}

func (e *RuleStateError) Error() string {
	return "unable to keep file '" + e.File.Path + "': " + e.Reason.String()
}

func CanKeepFileForError(err error) bool {
	if err, ok := err.(*RuleStateError); ok {
		switch err.Reason {
		case RuleStateErrorSizeTooSmall:
			return false
		}
	}
	return true
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

// FilesByFolder represents
type FilesByFolder map[string][]File

// FilesByDateAsc stores a slice of files, which should be sorted by date,
// from older to earlier
type FilesByDateAsc []File

func (files FilesByDateAsc) Len() int {
	return len(files)
}
func (files FilesByDateAsc) Less(i, j int) bool {
	return files[i].Date.Before(files[j].Date)
}
func (files FilesByDateAsc) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

// Sort sorts the files by date (asc)
func (files FilesByDateAsc) Sort() {
	sort.Sort(files)
}

// Sort sorts the files by date (asc)
func (files FilesByDateAsc) Reverse() {
	sort.Sort(sort.Reverse(files))
}

// Latest returns the most recent file from the list
func (files FilesByDateAsc) Latest() File {
	// sort the backups by desc
	files.Reverse()

	now := time.Now()
	for _, f := range files {
		if f.Date.After(now) {
			continue
		}
		return f
	}

	// return the latest element
	s := []File(files)
	return s[len(s)-1]
}

type SelectedFile struct {
	File
	Expiration time.Time
	Error      error
}

// FilesByDateAsc stores a slice of files, which should be sorted by date,
// from older to earlier
type SelectedFilesByExpirationAsc []SelectedFile

func (files SelectedFilesByExpirationAsc) Len() int {
	return len(files)
}
func (files SelectedFilesByExpirationAsc) Less(i, j int) bool {
	return files[i].Expiration.Before(files[j].Expiration)
}
func (files SelectedFilesByExpirationAsc) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

// Sort sorts the files by date (asc)
func (files SelectedFilesByExpirationAsc) Sort() {
	sort.Sort(files)
}

// Sort sorts the files by date (asc)
func (files SelectedFilesByExpirationAsc) Reverse() {
	sort.Sort(sort.Reverse(files))
}

// Latest returns the most recent file from the list
func (files SelectedFilesByExpirationAsc) Latest() SelectedFile {
	// sort the backups
	files.Sort()
	// return the latest element
	s := []SelectedFile(files)
	return s[len(s)-1]
}
