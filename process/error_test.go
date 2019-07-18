package process

// import (
// 	"testing"

// 	"github.com/agence-webup/backr/manager"
// )

// func TestSetError(t *testing.T) {

// 	sampleErrCollector := errorCollector{
// 		project: manager.Project{
// 			Name: "project1",
// 		},
// 		errorsToNotify: map[manager.RuleStateErrorType]map[manager.RuleID]manager.RuleStateError{},
// 	}

// 	rule := manager.Rule{Count: 3, MinAge: 1}
// 	errType := manager.RuleStateErrorObsolete
// 	err := manager.RuleStateError{
// 		File:   manager.File{Path: "project1/samplefile.tar.gz", Size: 300},
// 		Reason: errType,
// 	}

// 	// call SetError on an empty errorCollector
// 	sampleErrCollector.SetError(rule, err)

// 	errorsByType, ok := sampleErrCollector.errorsToNotify[errType]
// 	if !ok {
// 		t.Errorf("SetError should create a new map when error doesn't exist for this reason yet")
// 	}
// 	savedErr, ok := errorsByType[rule.GetID()]
// 	if !ok {
// 		t.Errorf("SetError must set the error according to the type and rule ID")
// 	}
// 	if err.File.Path != savedErr.File.Path || err.Reason != savedErr.Reason {
// 		t.Errorf("the error is not the one passed in the parameters")
// 	}

// 	newErr := manager.RuleStateError{
// 		File:   manager.File{Path: "project1/samplefile2.tar.gz", Size: 200},
// 		Reason: errType,
// 	}

// 	// SetError must keep the first set error for the same rule and error type
// 	// and ignore the new sent error
// 	sampleErrCollector.SetError(rule, newErr)

// 	ignoredErr := sampleErrCollector.errorsToNotify[errType][rule.GetID()]
// 	if ignoredErr.File.Path == newErr.File.Path {
// 		t.Errorf("SetError should not replace the existing error for the same rule and reason")
// 	}
// }
