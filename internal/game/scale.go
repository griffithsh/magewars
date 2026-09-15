package game

import "github.com/griffithsh/magewars/ecs"

// Scale defines a multiplier that is applied to the Entity's size.
type Scale struct {
	X, Y float64
}

// Name of this Component.
func (Scale) Name() ecs.ComponentName {
	return "Scale"
}
