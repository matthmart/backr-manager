package manager

import "fmt"

import "crypto/sha1"

// Notifier defines methods required to notify alerts
type Notifier interface {
	Notify(statement ProjectErrorStatement) error
}

// ProjectErrorStatement stores global error state for a project
type ProjectErrorStatement struct {
	Project  Project
	Count    int
	Reasons  map[RuleStateErrorType]string
	MaxLevel AlertLevel
}

// GetUniqueID returns an ID representing the statement
func (stmt *ProjectErrorStatement) GetUniqueID() string {
	str := fmt.Sprintf("n:%vc:%dr:%vl:%d", stmt.Project.Name, stmt.Count, stmt.Reasons, stmt.MaxLevel)
	d := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", d)
}

// AlertLevel represents a level of alert
type AlertLevel int

const (
	// Warning is a level to catch attention, but with no impact of the process
	Warning AlertLevel = iota
	// Critic is a level sent when the issue requires an action
	Critic
)

func (a *AlertLevel) String() string {
	if a != nil {
		switch *a {
		case Critic:
			return "critic"
		case Warning:
			return "warning"
		}
	}
	return "ok"
}
