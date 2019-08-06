package process

import (
	"fmt"
	"sort"
	"time"

	"github.com/agence-webup/backr/manager"
	"github.com/rs/zerolog/log"
)

// Execute runs the following process, for a specific reference date:
//   - fetch all projects
//   - fetch all files
//   - for each project and each rule of the project:
//     - check if a backup is needed
//     - if needed, determine the files fulfilling the rule
//     - if some files don't fulfill exactly the rule, an error is associated to the file, for this rule
//     - if backup is needed but no file is available, an error is set to the rule
//     - if some files are not needed anymore, by any rule, they are deleted, except if this prevents to fulfill the rule
func Execute(referenceDate time.Time, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository) error {
	pm := processManager{
		referenceDate: referenceDate,
		projectRepo:   projectRepo,
		fileRepo:      fileRepo,
	}

	err := pm.execute()

	return err
}

// Notify is responsible to send alerts, according to the state of each projects.
// If an error is associated to a rule or a file linked to a rule, an alert will be sent
func Notify(projectRepo manager.ProjectRepository, notifier manager.Notifier) error {
	projects, err := projectRepo.GetAll()
	if err != nil {
		log.Fatal().AnErr("error", err).Msg("unable to fetch all projects")
		return err
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
			if ruleState.Error != nil {
				projectErr.Count++
				projectErr.Reasons[ruleState.Error.Reason] = true
				projectErr.Level = manager.Critic
			}

			var firstErr *manager.RuleStateError

			files := manager.SelectedFilesSortedByExpirationDateDesc(ruleState.Files)
			for i, f := range files {
				if f.Error != nil {
					projectErr.Count++
					projectErr.Reasons[f.Error.Reason] = true
					if i == 0 {
						firstErr = f.Error
					}
				}
			}

			if firstErr != nil && firstErr.Reason == manager.RuleStateErrorObsolete {
				projectErr.Level = manager.Critic
			}

		}

		if projectErr.Count > 0 {
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

	return nil
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

	// to track if a file selection has been done
	hasPerformedSelection := false

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
			log.Info().Str("project", project.Name).Str("rule_id", string(rule.GetID())).Time("next_date", *ruleState.Next).Msg("backup needed. selecting files...")

			pm.selectFilesToBackup(&ruleState, filesByDateDesc)
			hasPerformedSelection = true
		} else {
			// logging
			if ruleState.Next == nil {
				log.Info().Str("project", project.Name).Str("rule_id", string(rule.GetID())).Msg("backup not needed. Next date is not set yet.")
			} else {
				log.Info().Str("project", project.Name).Str("rule_id", string(rule.GetID())).Time("next_date", *ruleState.Next).Time("ref_date", pm.referenceDate).Msg("backup not needed")
			}
		}

		// set the next backup date for fresh state
		if ruleState.Next == nil {
			n := pm.referenceDate.Add(1 * time.Hour * 24)
			ruleState.Next = &n
			log.Info().Time("next_date", n).Msg("set Next date")
		}

		// update state
		project.UpdateState(id, ruleState)

		// save project & state
		pm.projectRepo.Save(*project)
	}

	// remove unused files, only if a file selection has been done
	if hasPerformedSelection {
		filesToRemove := pm.getFilesToRemove(project, files, pm.referenceDate)
		log.Info().Str("project", project.Name).Int("count", len(filesToRemove)).Msg("files to be removed")

		for _, f := range filesToRemove {
			err := pm.fileRepo.RemoveFile(f)
			if err != nil {
				log.Error().Str("project", project.Name).Str("path", f.Path).Msg("unable to remove file")
				return fmt.Errorf("unable to remove file: %v", err)
			}
		}

		// save the state after removal
		project.RemoveFilesFromState(filesToRemove)
		// save project & state
		pm.projectRepo.Save(*project)
	}

	// fmt.Println("")
	// project.DebugPrint()
	// fmt.Println("")

	return nil
}

func (pm *processManager) selectFilesToBackup(ruleState *manager.RuleState, files []manager.File) {
	// olderRefDate allows to go back to the past to collect
	// previous files, if needed
	// the olderRefDate will be decremented by minAge for each file iteration
	olderRefDate := pm.referenceDate

	if len(files) == 0 {
		log.Debug().Caller().Msg("no file available")
		err := manager.RuleStateError{
			Reason: manager.RuleStateErrorNoFile,
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

		log.Debug().Caller().Int("count", len(files)).Msgf("available files count")
		log.Debug().Caller().Int("count", len(existingFilesByExpDesc)).Msg("existing files count")

		// iterate on each file
		for i, f := range files {

			// keep the most recent file just older than the ref data
			// so if the file's date is after the current reference date, skip the file
			// this allows to ignore files that don't fit with the period between each file
			// i.e. minAge: 3 => if we keep the 'today file', we want to keep the '3 days before file'
			// and not the 'yesterday file'
			if f.Date.After(olderRefDate) {
				log.Debug().Caller().Time("date", f.Date).Time("ref_date", olderRefDate).Str("path", f.Path).Msg("file date is after ref date")
				continue
			}

			log.Debug().Caller().Time("date", f.Date).Time("ref_date", olderRefDate).Str("path", f.Path).Msg("candidate file")

			// prepare the expiration date of the file
			expiration := f.Date.Add(time.Duration(ruleState.Rule.MinAge) * 24 * time.Hour)

			var fileError *manager.RuleStateError

			// check the size
			previousSize := int64(0)
			if i < len(files)-1 {
				previousSize = files[i+1].Size
			}
			if previousSize > 0 {
				acceptableSize := int64(float64(previousSize) * 0.5) // 50%
				if f.Size <= acceptableSize {
					err := manager.RuleStateError{
						File:   f,
						Reason: manager.RuleStateErrorSizeTooSmall,
					}
					fileError = &err
					log.Debug().Caller().Int64("previous_size", previousSize).Int64("actual_size", f.Size).Str("path", f.Path).Msg("file is at least 50% smaller than previous backup")
				}
			}

			// check if file is expired
			if expiration.Before(olderRefDate) {
				err := manager.RuleStateError{
					File:   f,
					Reason: manager.RuleStateErrorObsolete,
				}
				fileError = &err
				log.Debug().Caller().Time("ref_date", olderRefDate).Time("expiration", expiration).Str("path", f.Path).Msg("file is obsolete")
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
				log.Debug().Caller().Str("path", f.Path).Msg("new file, adding it to state")
			} else {
				// if it's already in the kept files, update the eventual error
				if i, ok := existingFilesIndexesByPath[f.Path]; ok && fileError != nil {
					ruleState.Files[i].Error = fileError
					log.Debug().Caller().Str("path", f.Path).Msg("existing file, update the associated error")
				}
			}

			// update Next date
			if fileError == nil {
				unit := 24 * time.Hour
				// unit := 1 * time.Minute
				next := f.Date.Add(time.Duration(ruleState.Rule.MinAge) * unit)
				if ruleState.Next == nil || next.After(*ruleState.Next) {
					l := log.Debug().Caller()
					if ruleState.Next != nil {
						l = l.Time("previous_next", *ruleState.Next)
					} else {
						l = l.Str("previous_next", "nil")
					}
					l.Time("new_next", next).Str("rule_id", string(ruleState.Rule.GetID())).Msg("Next date updated")
					ruleState.Next = &next
				}
			} else {
				log.Debug().Caller().Str("path", f.Path).Msg("Next date not updated: file has an error")
			}

			if fileError != nil && fileError.Reason == manager.RuleStateErrorSizeTooSmall {
				// don't update the refDate, trying to find another file to fulfill the needs of the rule
				log.Debug().Caller().Str("rule_id", string(ruleState.Rule.GetID())).Str("path", f.Path).Msg("file is too small, trying to find another file for the rule")
			} else {
				// substract (minAge * 24h)
				newRefDate := olderRefDate.Add(time.Duration(-ruleState.Rule.MinAge) * 24 * time.Hour)
				log.Debug().Caller().Time("older_ref_date", olderRefDate).Time("new_ref_date", newRefDate).Str("rule_id", string(ruleState.Rule.GetID())).Msg("decrease reference date")
				olderRefDate = newRefDate
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

			// we want to keep at least the rule count, and if they expire later than reference date, we keep them too
			if fileKeptCount < rs.Rule.Count || maxExpiration.After(referenceDate) {
				filesToKeep[f.Path] = true

				// we are keeping too small files, but we don't want them to count into the reliable backups
				if pm.canKeepFileForError(f.Error) {
					fileKeptCount++
				}
			}
		}
	}

	log.Info().Str("project", project.Name).Int("count", len(filesToKeep)).Msg("files to keep")

	filesToRemove := []manager.File{}
	for _, f := range allFiles {
		if _, ok := filesToKeep[f.Path]; !ok {
			filesToRemove = append(filesToRemove, f)
		}
	}

	return filesToRemove
}

func (pm *processManager) canKeepFileForError(err *manager.RuleStateError) bool {
	if err != nil {
		switch err.Reason {
		case manager.RuleStateErrorSizeTooSmall:
			return false
		}
	}
	return true
}
