package process

import (
	"fmt"
	"sort"
	"time"

	"github.com/agence-webup/backr/manager"
	"github.com/rs/zerolog/log"
)

func Execute(referenceDate time.Time, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository) error {
	pm := processManager{
		referenceDate: referenceDate,
		projectRepo:   projectRepo,
		fileRepo:      fileRepo,
	}

	err := pm.execute()

	return err
}

func Notify(projectRepo manager.ProjectRepository, notifier manager.Notifier) {
	projects, err := projectRepo.GetAll()
	if err != nil {
		log.Fatal().AnErr("error", err).Msg("unable to fetch all projects")
	}

	type projectError struct {
		Count   int
		Reasons map[manager.RuleStateErrorType]bool
		Level   manager.AlertLevel
	}

	for _, project := range projects {

		projectErr := projectError{Reasons: map[manager.RuleStateErrorType]bool{}}

		for _, ruleState := range project.State {

			// check for a global error
			if err, ok := ruleState.Error.(*manager.RuleStateError); ok {
				projectErr.Count++
				projectErr.Reasons[err.Reason] = true
				projectErr.Level = manager.Critic
			}

			var firstErr *manager.RuleStateError
			validFilesCount := 0

			files := manager.SelectedFilesSortedByExpirationDateDesc(ruleState.Files)
			for i, f := range files {
				if err, ok := f.Error.(*manager.RuleStateError); ok {
					projectErr.Count++
					projectErr.Reasons[err.Reason] = true
					if i == 0 {
						firstErr = err
					}
				} else {
					validFilesCount++
				}
			}

			if firstErr != nil && firstErr.Reason == manager.RuleStateErrorObsolete {
				projectErr.Level = manager.Critic
			}

		}

		if projectErr.Count > 0 {
			// lvl := manager.Warning
			// if projectErr.Reasons[manager.RuleStateErrorObsolete] {
			// 	lvl = manager.Critic
			// }
			reasonsLabels := []string{}
			for reason := range projectErr.Reasons {
				reasonsLabels = append(reasonsLabels, reason.String())
			}

			alert := manager.Alert{
				Title:   "Backup issue",
				Level:   projectErr.Level,
				Message: fmt.Sprintf("The project has %d error(s).", projectErr.Count),
				Metadata: map[string]interface{}{
					"project": project.Name,
					"count":   projectErr.Count,
					"reasons": reasonsLabels,
				},
			}

			notifier.Send(alert)
		}

	}
}

type processManager struct {
	referenceDate time.Time
	projectRepo   manager.ProjectRepository
	fileRepo      manager.FileRepository
}

func (pm *processManager) execute() error {
	projects, err := pm.projectRepo.GetAll()
	if err != nil {
		log.Fatal().AnErr("error", err).Msg("unable to fetch all projects")
	}

	// fetch backups
	filesByFolder, err := pm.fileRepo.GetAllByFolder()
	if err != nil {
		log.Fatal().AnErr("error", err).Msg("unable to fetch files from S3")
	}

	// process for each project
	for _, project := range projects {

		err := pm.processForProject(&project, filesByFolder)
		if err != nil {
			return err
		}

	}

	return nil
}

func (pm *processManager) processForProject(project *manager.Project, filesByFolder manager.FilesByFolder) error {
	// sort the rules
	rulesByMinAgeDesc := manager.RulesByMinAge(project.Rules)
	sort.Sort(sort.Reverse(rulesByMinAgeDesc))

	// get project files
	files := filesByFolder[project.Name]

	// sort files by date (desc)
	filesByDateDesc := manager.FilesSortedByDateDesc(files)

	// process each rule
	for _, rule := range rulesByMinAgeDesc {
		id := rule.GetID()

		// check if the state already exists
		ruleState, ok := project.State[id]
		if !ok {
			// if not, create it
			ruleState = manager.RuleState{
				Rule:  rule,
				Files: []manager.SelectedFile{},
			}
		}

		// check if a backup is wanted by the rule
		backupIsNeeded := ruleState.Check(pm.referenceDate)
		if backupIsNeeded {
			pm.selectFilesToBackup(&ruleState, filesByDateDesc)
		}

		// set the next backup date for fresh state
		if ruleState.Next == nil {
			n := pm.referenceDate.Add(1 * time.Hour * 24)
			ruleState.Next = &n
		}

		// update state
		project.UpdateState(id, ruleState)

		// save project & state
		pm.projectRepo.Save(*project)
	}

	// remove unused files
	filesToRemove := pm.getFilesToRemove(project, files, pm.referenceDate)
	fmt.Printf("[%v] needs to remove: %v\n", project.Name, filesToRemove)

	for _, f := range filesToRemove {
		err := pm.fileRepo.RemoveFile(f)
		if err != nil {
			return fmt.Errorf("unable to remove file: %v", err)
		}
	}

	// save the state after removal
	project.RemoveFilesFromState(filesToRemove)

	fmt.Println("")
	project.DebugPrint()
	fmt.Println("")

	return nil
}

func (pm *processManager) selectFilesToBackup(ruleState *manager.RuleState, files []manager.File) {
	// olderRefDate allows to go back to the past to collect
	// previous files, if needed
	// the olderRefDate will be decremented by minAge for each file iteration
	olderRefDate := pm.referenceDate

	if len(files) == 0 {
		log.Debug().Str("func", "selectFilesToBackup").Msg("no file available")
		err := manager.RuleStateError{
			RuleState: *ruleState,
			Reason:    manager.RuleStateErrorNoFile,
		}
		ruleState.Error = &err
	} else {

		// reset the error, if any
		ruleState.Error = nil

		existingFilesIndexesByPath := map[string]int{}
		for i, f := range ruleState.Files {
			existingFilesIndexesByPath[f.Path] = i
		}

		// fetch already kept files and sort them by expiration date (desc)
		existingFilesByPath := map[string]manager.SelectedFile{}
		existingFilesByExpDesc := manager.SelectedFilesSortedByExpirationDateDesc(ruleState.Files)
		for _, f := range existingFilesByExpDesc {
			existingFilesByPath[f.Path] = f
		}

		log.Debug().Str("func", "selectFilesToBackup").Int("count", len(files)).Msgf("available files count")
		log.Debug().Str("func", "selectFilesToBackup").Int("count", len(existingFilesByExpDesc)).Msg("existing files count")

		// iterate on each file
		for i, f := range files {

			// keep the most recent file just older than the ref data
			// so if the file's date is after the current reference date, skip the file
			// this allows to ignore files that don't fit with the period between each file
			// i.e. minAge: 3 => if we keep the 'today file', we want to keep the '3 days before file'
			// and not the 'yesterday file'
			if f.Date.After(olderRefDate) {
				log.Debug().Str("func", "selectFilesToBackup").Str("date", f.Date.String()).Str("ref_date", olderRefDate.String()).Str("path", f.Path).Msg("file date is after ref date")
				continue
			}

			log.Debug().Str("func", "selectFilesToBackup").Str("date", f.Date.String()).Str("ref_date", olderRefDate.String()).Str("path", f.Path).Msg("candidate file")

			// prepare the expiration date of the file
			expiration := f.Date.Add(time.Duration(ruleState.Rule.MinAge) * 24 * time.Hour)

			var fileError error

			// check the size
			previousSize := int64(0)
			if i < len(files)-1 {
				previousSize = files[i+1].Size
			}
			if previousSize > 0 {
				acceptableSize := int64(float64(previousSize) * 0.5) // 50%
				if f.Size <= acceptableSize {
					err := manager.RuleStateError{
						RuleState: *ruleState,
						File:      f,
						Reason:    manager.RuleStateErrorSizeTooSmall,
					}
					fileError = &err
				}
			}

			// check if file is expired
			if expiration.Before(olderRefDate) {
				err := manager.RuleStateError{
					RuleState: *ruleState,
					File:      f,
					Reason:    manager.RuleStateErrorObsolete,
				}
				fileError = &err
			}

			if fileError != nil {
				log.Debug().Caller().AnErr("err", fileError).Str("path", f.Path).Msg("detected file error")
			} else {
				log.Debug().Caller().Str("path", f.Path).Msg("no file error detected")
			}

			// keep the file, updating the state
			var selectedFile *manager.SelectedFile
			if _, ok := existingFilesByPath[f.Path]; !ok {
				// if not, append it to the kept files in state
				selectedFile = &manager.SelectedFile{
					File:       f,
					Expiration: expiration,
					Error:      fileError,
				}
				ruleState.Files = append(ruleState.Files, *selectedFile)
			} else {
				// if it's already in the kept files, update the eventual error
				if i, ok := existingFilesIndexesByPath[f.Path]; ok && fileError != nil {
					ruleState.Files[i].Error = fileError
				}
			}

			// update Next date
			if fileError == nil {
				unit := 24 * time.Hour
				// unit := 1 * time.Minute
				next := f.Date.Add(time.Duration(ruleState.Rule.MinAge) * unit)
				if ruleState.Next == nil || next.After(*ruleState.Next) {
					ruleState.Next = &next
				}
			}

			if err, ok := fileError.(*manager.RuleStateError); ok && err.Reason == manager.RuleStateErrorSizeTooSmall {
				// don't update the refDate, trying to find another file to fulfill the needs of the rule
			} else {
				// substract (minAge * 24h)
				olderRefDate = olderRefDate.Add(time.Duration(-ruleState.Rule.MinAge) * 24 * time.Hour)
			}
		}
	}
}

func (pm *processManager) getFilesToRemove(project *manager.Project, allFiles []manager.File, referenceDate time.Time) []manager.File {

	// stores in a map the most recent expiration date for each file associated to the project
	// a file can be shared by several rules, but the expiration date can be different for each one.
	// So we walk across each rule to know the most recent expiration date and store it into a map.
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

		filesByExpDateDesc := manager.SelectedFilesSortedByExpirationDateDesc(rs.Files)

		fileKeptCount := 0
		for _, f := range filesByExpDateDesc {

			maxExpiration := filesMaxExpiration[f.Path]
			// if maxExpiration.Before(now) && fileKeptCount > rs.Rule.Count {
			// 	filesToKeep[f.Path] = false
			// } else {
			// 	filesToKeep[f.Path] = true
			// 	fileKeptCount++
			// }
			fmt.Printf("%v: (fileKeptCount(%v) < rs.Rule.Count(%v) || maxExp(%v).After(%v))  && CanKeepFileForError(%v)\n", f.Path, fileKeptCount, rs.Rule.Count, maxExpiration, referenceDate, pm.canKeepFileForError(f.Error))

			// TODO: vérifier si 'CanKeepFileForError' est nécessaire dans le if
			// we are keeping too small files, but we don't want them to count into the reliable backups
			if fileKeptCount < rs.Rule.Count || maxExpiration.After(referenceDate) /* && pm.canKeepFileForError(f.Error)*/ {
				filesToKeep[f.Path] = true

				// fmt.Printf("  canKeepFile(%v)\n", CanKeepFileForError(f.Error))
				if pm.canKeepFileForError(f.Error) {
					fileKeptCount++
				}
			}
		}
		// }
	}

	fmt.Printf("[%v] files to keep: %+v\n", project.Name, filesToKeep)

	filesToRemove := []manager.File{}
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

func (pm *processManager) canKeepFileForError(err error) bool {
	if err, ok := err.(*manager.RuleStateError); ok {
		switch err.Reason {
		case manager.RuleStateErrorSizeTooSmall:
			return false
		}
	}
	return true
}
