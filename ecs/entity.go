package ecs

import "math/rand"

type Entity uint64

// CreateEntity creates an Entity
func (w *World) CreateEntity() Entity {
	try := Entity(rand.Int63())
	// Is the try already in use? This is hugely unlikely given there 2^63 possibilities.
	if _, ok := w.entities[try]; ok {
		// Recurse to try again.
		return w.CreateEntity()
	}
	w.entities[try] = struct{}{}
	return try
}

// DestroyEntity removes an Entity and all its Components.
func (w *World) DestroyEntity(e Entity) {
	if dependants, ok := w.dependencies[e]; ok {
		for _, e := range dependants {
			w.DestroyEntity(e)
		}
		delete(w.dependencies, e)
	}

	for _, entities := range w.components {
		delete(entities, e)
	}
	delete(w.entities, e)
}
