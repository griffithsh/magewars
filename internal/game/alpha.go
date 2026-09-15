package game

import "github.com/griffithsh/magewars/ecs"

// Alpha is a component that controls how visible an Entity should be rendered.
type Alpha struct {
	Value float64 // Value should be between 0.0 and 1.0
}

// Name of this Component.
func (Alpha) Name() ecs.ComponentName {
	return "Alpha"
}
