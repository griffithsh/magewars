package game

import "github.com/griffithsh/magewars/ecs"

// SpriteRepeat is a component that repeats (or "tiles") a sprite across a width
// and height.
type SpriteRepeat struct {
	W, H int
}

// Type of this Component.
func (SpriteRepeat) Name() ecs.ComponentName {
	return "SpriteRepeat"
}
