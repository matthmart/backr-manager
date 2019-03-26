package process

import (
	"fmt"
	"sort"
	"time"

	"github.com/agence-webup/backr/manager"
	"github.com/rs/zerolog/log"
)

type rState struct {
	// SelectedFiles []manager.File
	PreviousFile *manager.File
}

// func (rs *rState) GetFileSpec(refDate time.Time, count int) (*time.Time, *time.Time) {

// }

// func Execute(referenceDate time.Time, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository, notifier manager.Notifier) error {
// 	projects, err := projectRepo.GetAll()
// 	if err != nil {
// 		log.Fatal().AnErr("error", err).Msg("unable to fetch all projects")
// 	}

// 	// fmt.Println(projects)

// 	// fetch backups
// 	filesByFolder, err := fileRepo.GetAllByFolder()
// 	if err != nil {
// 		log.Fatal().AnErr("error", err).Msg("unable to fetch files from S3")
// 	}

// 	// now := time.Now()
// 	// today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

// 	// process for each project
// 	for i, project := range projects {

// 		// sort the rules
// 		rulesByMinAgeDesc := manager.RulesByMinAge(project.Rules)
// 		sort.Sort(sort.Reverse(rulesByMinAgeDesc))

// 		// get project files
// 		files, _ := filesByFolder[project.Name]

// 		filesByDateDesc := manager.FilesByDateAsc(files)
// 		filesByDateDesc.Reverse()

// 		// DEBUG
// 		fmt.Println("files:")
// 		for _, f := range files {
// 			fmt.Printf(" - %v\n", f.Path)
// 		}

// 		filesToKeep := map[string]bool{}

// 		// process each rule
// 		for _, rule := range rulesByMinAgeDesc {
// 			id := rule.GetID()
// 			// check if the state already exists
// 			ruleState, ok := project.State[id]
// 			if !ok {
// 				ruleState = manager.RuleState{
// 					Rule:  rule,
// 					Files: []manager.File{},
// 				}
// 			}

// 			refDate := time.Date(referenceDate.Year(), referenceDate.Month(), referenceDate.Day(), 0, 0, 0, 0, time.Local)

// 			// try to collect every possible file
// 			count := int(math.Min(float64(len(files)), float64(rule.Count)))
// 			// var previousDate *time.Time
// 			for i := 0; i < count; i++ {
// 				for _, file := range filesByDateDesc {

// 					// if previousDate != nil {
// 					// 	refDate = *previousDate
// 					// }

// 					lowerDate := refDate.Add(time.Duration(-i*rule.MinAge) * time.Hour * 24)
// 					upperDate := lowerDate.Add(24 * time.Hour)

// 					fmt.Printf("   -> rule_id:%v lower:%v file:[%v / %v]", id, lowerDate, file.Path, file.Date)
// 					if file.Date.After(lowerDate) && file.Date.Before(upperDate) {
// 						fmt.Printf(" => kept\n")
// 						// ruleState.Keep(file)
// 						filesToKeep[file.Path] = true
// 						// previousDate = &refDate
// 						break
// 					}
// 					fmt.Println("")
// 				}
// 			}

// 			// backupIsNeeded := ruleState.Check(referenceDate)
// 			// if backupIsNeeded {
// 			// 	if !fileExists {
// 			// 		notifier.Send(manager.Alert{
// 			// 			Title:   "No backup found",
// 			// 			Message: "The project folder in the bucket does not exist.",
// 			// 			Level:   manager.Critic,
// 			// 			Metadata: map[string]string{
// 			// 				"project": project.Name,
// 			// 			},
// 			// 		})
// 			// 		continue
// 			// 	}

// 			// 	// get the most recent file
// 			// 	latest := manager.FilesByDateAsc(files).Latest()
// 			// 	// and keep it (it must not be deleted)
// 			// 	err := ruleState.Keep(latest)

// 			// 	if err, ok := err.(*manager.RuleStateError); ok {
// 			// 		notifier.Send(manager.Alert{
// 			// 			Title:   "Error with latest file",
// 			// 			Message: err.Error(),
// 			// 			Level:   manager.Warning,
// 			// 			Metadata: map[string]string{
// 			// 				"project": project.Name,
// 			// 			},
// 			// 		})
// 			// 	}
// 			// }

// 			// clear old backups
// 			ruleState.Clear()

// 			// update state
// 			project.UpdateState(id, ruleState)

// 			// TODO: save project & state
// 			projects[i] = project
// 			projectRepo.Save(project)
// 		}

// 		// files to keep
// 		fmt.Printf("files to keep: %v\n", filesToKeep)

// 		// // remove unused files
// 		// filesToRemove := project.GetFilesToRemove(files)
// 		// fmt.Printf("[%v] needs to remove: %v\n", project.Name, filesToRemove)

// 		// for _, f := range filesToRemove {
// 		// 	err := fileRepo.RemoveFile(f)
// 		// 	if err != nil {
// 		// 		return fmt.Errorf("unable to remove file: %v", err)
// 		// 	}
// 		// }

// 		// files, ok := filesByFolder[project.Name]
// 		// if !ok {
// 		// 	notifier.Send(manager.Alert{
// 		// 		Title:   "No backup found",
// 		// 		Message: "The project folder in the bucket does not exist.",
// 		// 		Level:   manager.Critic,
// 		// 		Metadata: map[string]string{
// 		// 			"project": project.Name,
// 		// 		},
// 		// 	})
// 		// 	continue
// 		// }

// 		// fmt.Println(files)
// 	}

// 	// fmt.Println(projects)

// 	return nil
// }

func Execute(referenceDate time.Time, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository, notifier manager.Notifier) error {
	projects, err := projectRepo.GetAll()
	if err != nil {
		log.Fatal().AnErr("error", err).Msg("unable to fetch all projects")
	}

	// fetch backups
	filesByFolder, err := fileRepo.GetAllByFolder()
	if err != nil {
		log.Fatal().AnErr("error", err).Msg("unable to fetch files from S3")
	}

	// process for each project
	for i, project := range projects {

		// sort the rules
		rulesByMinAgeDesc := manager.RulesByMinAge(project.Rules)
		sort.Sort(sort.Reverse(rulesByMinAgeDesc))

		// get project files
		files := filesByFolder[project.Name]

		// sort files by date (desc)
		filesByDateDesc := manager.FilesByDateAsc(files)
		filesByDateDesc.Reverse()

		// prepare map & function to collect errors to notify
		errorsToNotify := map[manager.RuleStateErrorType]map[manager.RuleID]manager.RuleStateError{}
		setError := func(rule manager.Rule, err manager.RuleStateError) {
			ruleID := rule.GetID()
			if _, ok := errorsToNotify[err.Reason]; !ok {
				errorsToNotify[err.Reason] = map[manager.RuleID]manager.RuleStateError{}
			}
			// keep only the first one set error
			if _, ok := errorsToNotify[err.Reason][ruleID]; !ok {
				errorsToNotify[err.Reason][ruleID] = err
			}
		}

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
			backupIsNeeded := ruleState.Check(referenceDate)
			if backupIsNeeded {

				// olderRefDate allows to go back to the past to collect
				// previous files, if needed
				// the olderRefDate will be decremented by minAge for each file iteration
				olderRefDate := referenceDate

				if len(filesByDateDesc) == 0 {
					err := manager.RuleStateError{
						RuleState: ruleState,
						Reason:    manager.RuleStateErrorObsolete,
					}
					setError(rule, err)
				} else {

					// fetch already kept files and sort them by expiration date (desc)
					existingFilesByPath := map[string]manager.SelectedFile{}
					existingFilesByExpDesc := manager.SelectedFilesByExpirationAsc(ruleState.Files)
					existingFilesByExpDesc.Reverse()
					for _, f := range existingFilesByExpDesc {
						existingFilesByPath[f.Path] = f

						// fmt.Printf("i(%v) >= len(existingFilesByExpDesc)(%v)\n", i, len(existingFilesByExpDesc))
						// if i >= len(existingFilesByExpDesc)-1 {
						// 	break
						// }
					}

					// iterate on each file
					for i, f := range filesByDateDesc {

						// if the file's date is after the current reference date, skip the file
						// this allows to ignore files that don't fit with the period between each file
						// i.e. minAge: 3 => if we keep the 'today file', we want to keep the '3 days before file'
						// and not the 'yesterday file'
						if f.Date.After(olderRefDate) {
							continue
						}

						var fileError error

						// check the size
						previousSize := int64(0)
						if i < len(filesByDateDesc)-1 {
							previousSize = filesByDateDesc[i+1].Size
						}
						if previousSize > 0 {
							acceptableSize := int64(float64(previousSize) * 0.5) // 50%
							if f.Size <= acceptableSize {
								err := manager.RuleStateError{
									RuleState: ruleState,
									File:      f,
									Reason:    manager.RuleStateErrorSizeTooSmall,
								}
								fileError = &err
								setError(rule, err)
							}
						}

						// check if file is expired
						if existingFile, ok := existingFilesByPath[f.Path]; ok {
							// only for the first file
							if existingFile.Expiration.Before(olderRefDate) && i == 0 {
								// return &RuleStateError{File: file, Reason: RuleStateErrorObsolete}
								err := manager.RuleStateError{
									RuleState: ruleState,
									File:      f,
									Reason:    manager.RuleStateErrorObsolete,
								}
								fileError = &err
								setError(rule, err)
							}
						}

						//
						ruleState.Keep(f, fileError)

						// substract (minAge * 24h)
						olderRefDate = olderRefDate.Add(time.Duration(-rule.MinAge) * 24 * time.Hour)
					}
				}
			}

			if ruleState.Next == nil {
				n := referenceDate.Add(1 * time.Hour * 24)
				ruleState.Next = &n
			}

			// update state
			project.UpdateState(id, ruleState)

			// TODO: save project & state
			projects[i] = project
			projectRepo.Save(project)
		}

		// notify
		for reason, errorByID := range errorsToNotify {
			// format info on the error
			formattedInfo := map[string]string{}
			for ruleID, err := range errorByID {
				formattedInfo[string(ruleID)] = err.File.Path
			}

			notifier.Send(manager.Alert{
				Title:   "Error with file",
				Message: reason.String(),
				Level:   manager.Warning,
				Metadata: map[string]interface{}{
					"project": project.Name,
					"reason":  reason,
					"info":    formattedInfo,
				},
			})
		}

		// remove unused files
		filesToRemove := project.GetFilesToRemove(files, referenceDate)
		fmt.Printf("[%v] needs to remove: %v\n", project.Name, filesToRemove)

		for _, f := range filesToRemove {
			err := fileRepo.RemoveFile(f)
			if err != nil {
				return fmt.Errorf("unable to remove file: %v", err)
			}
		}

		// save the state after removal
		project.RemoveFilesFromState(filesToRemove)

	}

	for _, p := range projects {
		p.DebugPrint()
	}

	return nil
}
