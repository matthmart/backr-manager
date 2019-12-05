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

func (n *basicNotifier) Notify(stmt manager.ProjectErrorStatement) error {
	fmt.Println("")

	switch stmt.MaxLevel {
	case manager.Warning:
		fmt.Println("*** ⚠️  WARNING ***")
	case manager.Critic:
		fmt.Println("*** 🆘  CRITICAL ***")
	}

	fmt.Printf("→ %s\n", stmt.Project.Name)
	j, _ := json.Marshal(stmt.Reasons)
	fmt.Println(string(j))
	fmt.Println("——————————————————————")
	fmt.Println("")

	return nil
}
