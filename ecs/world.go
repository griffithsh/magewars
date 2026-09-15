package ecs

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type World struct {
	entities     map[Entity]struct{}
	components   map[ComponentName]map[Entity]Component
	dependencies map[Entity][]Entity
	systems      map[string]System
}

func NewWorld() *World {
	return &World{
		entities:     map[Entity]struct{}{},
		components:   map[ComponentName]map[Entity]Component{},
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
			sb.WriteString(err.Error() + "\n")
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

// Get returns the list of entities that have all of the provided names.
func (w *World) Get(types ...ComponentName) []Entity {
	result := []Entity{}
	// Check every Entity ...
	for e := range w.entities {
		// For every type to check ...
		count := 0
		for _, ty := range types {
			// If there are no components for that type, break from checking this entity further.
			if _, ok := w.components[ty]; !ok {
				break
			}

			// If this entity is present in the components then add it to the result.
			if _, ok := w.components[ty][e]; ok {
				count++
			}
		}
		if len(types) == count {
			result = append(result, e)
		}
	}
	return result
}

// Single is like Get, but will return the default value for Entity and false if
// there is not exactly one Entity with the requested Components.
func (w *World) Single(types ...ComponentName) (Entity, bool) {
	es := w.Get(types...)
	if len(es) != 1 {
		var d Entity
		return d, false
	}
	return es[0], true
}

// Exists returns whether an entity exists in this World.
func (w *World) Exists(e Entity) bool {
	_, ok := w.entities[e]
	return ok
}

// Get retrieves the Component of Type T for Entity.
func Get[T Component](u *World, e Entity) *T {
	var dummy T
	name := dummy.Name()

	if v, ok := u.components[name]; ok {
		// if v, ok := v[e]; ok {
		// 	val := v.(*T)
		// 	return val
		// }
		if componentAny, ok := v[e]; ok {
			// 1. Assert the interface to the underlying concrete type 'T'
			val, ok := componentAny.(T)
			if !ok {
				return nil
			}
			// 2. Take the address of that concrete value to return '*T'
			return &val
		}
	}
	return nil
}

// Add a component to Entity e, replacing the existing coponent if one of that
// type already exists.
func (w *World) Add(e Entity, c Component) error {
	key := c.Name()
	// if !strings.HasPrefix(reflect.TypeOf(c).String(), "*") {
	// 	panic(fmt.Sprintf("cannot add non-pointer type %s", key))
	// }
	if _, ok := w.components[key]; !ok {
		w.components[key] = map[Entity]Component{}
	}

	w.components[key][e] = c
	return nil
}

// Remove the component with the specified type from the Entity.
func (w *World) Remove(e Entity, component Component) {
	// key := NameOf(component)
	key := component.Name()
	delete(w.components[key], e)

	// If that was the last of this type, then remove the map that stores them.
	if len(w.components[key]) == 0 {
		delete(w.components, key)
	}
}

// Clear all Entities and their Components from the World, resetting it to an empty state.
func (w *World) Clear() {
	w.entities = map[Entity]struct{}{}
	w.components = map[ComponentName]map[Entity]Component{}
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
