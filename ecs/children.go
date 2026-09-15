package ecs

import "time"

// ChildrenSystem manages Parent Components.
type ChildrenSystem struct {
}

// NewChildrenSystem constructs a new ParentSystem.
func NewChildrenSystem() *ChildrenSystem {
	return &ChildrenSystem{}
}

// Update the ChildrenSystem to remove references to Children that have been
// destroyed from the Parents that claim them.
func (ps *ChildrenSystem) Update(w *World, _ time.Duration) error {
	for _, e := range w.Get(Children{}.Name()) {
		children := Get[Children](w, e)
		val := []Entity{}
		for _, child := range children.Value {
			if !w.Exists(child) {
				continue
			}
			val = append(val, child)
		}
		children.Value = val
	}
	return nil
}

// Children Component allows Entities to keep a reference to other entities.
type Children struct {
	Value []Entity
}

// Type of this Component.
func (Children) Name() ComponentName {
	return "Children"
}
