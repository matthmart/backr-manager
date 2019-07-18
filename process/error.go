package process

// import "github.com/agence-webup/backr/manager"

// type errorCollector struct {
// 	project        manager.Project
// 	errorsToNotify map[manager.RuleStateErrorType]map[manager.RuleID]manager.RuleStateError
// }

// type errorCollectorByProjectName map[string]errorCollector

// func newErrorCollector(project manager.Project) errorCollector {
// 	collector := errorCollector{
// 		project:        project,
// 		errorsToNotify: map[manager.RuleStateErrorType]map[manager.RuleID]manager.RuleStateError{},
// 	}
// 	return collector
// }

// func (col *errorCollector) SetError(rule manager.Rule, err manager.RuleStateError) {
// 	ruleID := rule.GetID()
// 	if _, ok := col.errorsToNotify[err.Reason]; !ok {
// 		col.errorsToNotify[err.Reason] = map[manager.RuleID]manager.RuleStateError{}
// 	}
// 	// keep only the first one set error
// 	if _, ok := col.errorsToNotify[err.Reason][ruleID]; !ok {
// 		col.errorsToNotify[err.Reason][ruleID] = err
// 	}
// }

// func (col *errorCollector) Notify(notifier manager.Notifier) {
// 	for reason, errorByID := range col.errorsToNotify {
// 		// format info on the error
// 		formattedInfo := map[string]string{}
// 		for ruleID, err := range errorByID {
// 			formattedInfo[string(ruleID)] = err.File.Path
// 		}

// 		notifier.Send(manager.Alert{
// 			Title:   "Error with file",
// 			Message: reason.String(),
// 			Level:   manager.Warning,
// 			Metadata: map[string]interface{}{
// 				"project": col.project.Name,
// 				"reason":  reason,
// 				"info":    formattedInfo,
// 			},
// 		})
// 	}
// }
