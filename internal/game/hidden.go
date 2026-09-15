package game

import "github.com/griffithsh/magewars/ecs"

// Hidden is a Component that prevents the Entity from being rendered.
type Hidden struct{}

// Name of this Component.
func (f Hidden) Name() ecs.ComponentName {
	return "Hidden"
}
