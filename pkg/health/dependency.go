package health

import "fmt"

// Depender defines the interface for all concrete dependency implementations.
type Depender interface {
	Check() (map[string]interface{}, error) // Checks health, expects optional config/state map, and and error (nil if healthy).
}

// Dependency defines a registered dependency.
type Dependency struct {
	Name string   `json:"-"`
	Desc string   `json:"desc"`
	Item Depender `json:"item"`
	key  string   // Unique, as lowercase Name.
}

func (d *Dependency) String() string {
	return fmt.Sprintf(
		"{%T: %v (%v) @ %v}",
		d, d.Name, d.Desc, d.Item,
	)
}
