package manager

// Notifier defines methods required to notify alerts
type Notifier interface {
	Send(alert Alert)
}

// Alert stores notfication data
type Alert struct {
	Title    string
	Message  string
	Level    AlertLevel
	Metadata interface{}
}

// AlertLevel represents a level of alert
type AlertLevel int

const (
	// Warning is a level to catch attention, but with no impact of the process
	Warning AlertLevel = iota
	// Critic is a level sent when the issue requires an action
	Critic
)
