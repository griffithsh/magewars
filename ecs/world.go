package ecs

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type World struct {
	entities     map[Entity]struct{}
	components   map[reflect.Type]map[Entity]any
	dependencies map[Entity][]Entity
	systems      map[string]System
}

func NewWorld() *World {
	return &World{
		entities:     map[Entity]struct{}{},
		components:   map[reflect.Type]map[Entity]any{},
		dependencies: map[Entity][]Entity{},
		systems:      map[string]System{},
	}
}

func (w *World) Register(systems ...System) error {
	errs := []error{}
	for _, system := range systems {
		// We're trying to find a canonical, non-colliding name of the system
		// being registered to use as a unique identifier.
		name := strings.TrimLeft(reflect.TypeOf(system).String(), "*")

		if _, ok := w.systems[name]; ok {
			errs = append(errs, fmt.Errorf("duplicate system registered %q", name))
		}
		w.systems[name] = system
	}

	if len(errs) > 0 {
		sb := strings.Builder{}
		for _, err := range errs {
			sb.WriteString(fmt.Sprintf("%v\n", err))
		}
		return fmt.Errorf("%v", sb.String())
	}

	return nil
}

func (w *World) Update(elapsed time.Duration) error {
	for _, s := range w.systems {
		if err := s.Update(w, elapsed); err != nil {
			return err
		}
	}
	return nil
}

// Query returns the list of entities that have all of the provided names.
func (w *World) Query(types ...reflect.Type) []Entity {
	if len(types) == 0 {
		return nil
	}

	// Find the smallest matching component set first to minimize iterations
	var smallestMap map[Entity]any
	minLen := -1

	for _, t := range types {
		compMap, ok := w.components[t]
		if !ok || len(compMap) == 0 {
			return nil // If any component type doesn't exist, zero entities match
		}
		if minLen == -1 || len(compMap) < minLen {
			minLen = len(compMap)
			smallestMap = compMap
		}
	}

	var result []Entity

	// Filter entities against all requested component types
	for entity := range smallestMap {
		hasAll := true
		for _, t := range types {
			if _, ok := w.components[t][entity]; !ok {
				hasAll = false
				break
			}
		}
		if hasAll {
			result = append(result, entity)
		}
	}

	return result
}

// Exists returns whether an entity exists in this World.
func (w *World) Exists(e Entity) bool {
	_, ok := w.entities[e]
	return ok
}

// Add accepts a pointer to any value T
func Add[T any](w *World, e Entity, c *T) {
	t := reflect.TypeOf(c) // E.g., *Position

	if _, ok := w.components[t]; !ok {
		w.components[t] = map[Entity]any{}
	}

	w.components[t][e] = c
}

// Get returns *T directly from the map
func Get[T any](w *World, e Entity) *T {
	t := reflect.TypeOf((*T)(nil)) // Type representation of *T

	if v, ok := w.components[t]; ok {
		if componentAny, ok := v[e]; ok {
			return componentAny.(*T) // Safe pointer type assertion
		}
	}
	return nil
}

// Remove removes the component of type T from Entity e.
func Remove[T any](w *World, e Entity) {
	t := reflect.TypeOf((*T)(nil))

	compMap, ok := w.components[t]
	if !ok {
		return // Component type doesn't exist in the world, nothing to remove
	}

	// Delete the entity from this component type's sub-map
	delete(compMap, e)

	// Clean up the parent map if no entities have this component left
	if len(compMap) == 0 {
		delete(w.components, t)
	}
}

// Clear all Entities and their Components from the World, resetting it to an empty state.
func (w *World) Clear() {
	w.entities = map[Entity]struct{}{}
	w.components = map[reflect.Type]map[Entity]any{}
}

// Dependency adds a cascading delete rule for a pair of Entities. When parent
// is destroyed, then child is also destroyed.
func (w *World) Dependency(parent, child Entity) {
	w.dependencies[parent] = append(w.dependencies[parent], child)
}

// Len returns the number of Entites currently in the world.
func (w *World) Len() int {
	return len(w.entities)
}
