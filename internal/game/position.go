package game

import "github.com/griffithsh/magewars/ecs"

// Center defines the centre of something.
type Center struct {
	X, Y float64
}

// Position is a Component that anything with a position in the world has.
type Position struct {
	Center Center
	Layer  int

	// Absolute Positioned Components use screen coordinates, not world
	// coordinates.
	Absolute bool
}

// Name of this component.
func (p Position) Name() ecs.ComponentName {
	return "Position"
}
