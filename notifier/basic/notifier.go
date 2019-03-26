package basic

import (
	"encoding/json"
	"fmt"

	"github.com/agence-webup/backr/manager"
)

// NewNotifier returns a basic notifier
func NewNotifier() manager.Notifier {
	return &basicNotifier{}
}

type basicNotifier struct {
}

func (n *basicNotifier) Send(alert manager.Alert) {
	fmt.Println("")

	switch alert.Level {
	case manager.Warning:
		fmt.Println("*** ⚠️  WARNING ***")
	case manager.Critic:
		fmt.Println("*** 🆘  CRITICAL ***")
	}

	fmt.Printf("→ %s\n", alert.Title)
	fmt.Println(alert.Message)
	j, _ := json.Marshal(alert.Metadata)
	fmt.Println(string(j))
	fmt.Println("——————————————————————")
	fmt.Println("")

}
