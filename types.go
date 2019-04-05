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

		filesByExpDateDesc := SelectedFilesSortedByExpirationDateDesc(rs.Files)

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

			// TODO: vérifier si 'CanKeepFileForError' est nécessaire dans le if
			// => on veut garder les fichiers trop petits, mais pas qu'ils comptent dans les backups valables
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
			errState := "✅"
			if err, ok := f.Error.(*RuleStateError); ok {
				switch err.Reason {
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

	fmt.Printf("file error: %+v", someError)

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
	// if err, ok := err.(*RuleStateError); ok {
	// 	switch err.Reason {
	// 	case RuleStateErrorSizeTooSmall:
	// 		return false
	// 	}
	// }
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

// FilesSortedByDateAsc returns a slice of files,
// sorted by date from older to earlier
func FilesSortedByDateAsc(files []File) []File {
	f := make([]File, len(files))
	copy(f, files)

	sorted := sortedFilesByDate(f)
	sort.Sort(sorted)

	return sorted
}

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

// // Latest returns the most recent file from the list
// func (files FilesByDateAsc) Latest() File {
// 	// sort the backups by desc
// 	files.Reverse()

// 	now := time.Now()
// 	for _, f := range files {
// 		if f.Date.After(now) {
// 			continue
// 		}
// 		return f
// 	}

// 	// return the latest element
// 	s := []File(files)
// 	return s[len(s)-1]
// }

type SelectedFile struct {
	File
	Expiration time.Time
	Error      error
}

// SelectedFilesByExpirationDateDesc stores a slice of files (associated to a rule state),
// which should be sorted by expiration date, from older to earlier
type SelectedFilesByExpirationDateDesc []SelectedFile

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

// // Latest returns the most recent file from the list
// func (files SelectedFilesByExpirationAsc) Latest() SelectedFile {
// 	// sort the backups
// 	files.Sort()
// 	// return the latest element
// 	s := []SelectedFile(files)
// 	return s[len(s)-1]
// }
