package game

import (
	"time"

	"github.com/griffithsh/magewars/ecs"
	"github.com/griffithsh/magewars/event"
)

type CollisionReversionSystem struct {
	bus       *event.Bus
	events    []*CollisionEvent
	maxEvents int
}

func NewCollisionReversionSystem(bus *event.Bus) *CollisionReversionSystem {
	const maxEvents = 1000
	sys := CollisionReversionSystem{
		bus:       bus,
		events:    make([]*CollisionEvent, maxEvents/10),
		maxEvents: maxEvents,
	}
	event.Subscribe(bus, func(evt *CollisionEvent) {
		// Don't allocate without bounds!
		if len(sys.events) >= sys.maxEvents {
			panic("💀 too many collisions!")
		}
		sys.events = append(sys.events, evt)
	})
	return &sys
}

// getMass of an Entity, returning 0 when the Entity has no Mass Component.
func getMass(w *ecs.World, e ecs.Entity) float64 {
	mass := ecs.Get[Mass](w, e)

	if mass == nil {
		return 0
	}
	return mass.Value

}

func (s *CollisionReversionSystem) Update(w *ecs.World, t time.Duration) {
	for _, evt := range s.events {
		p1 := ecs.Get[Position](w, evt.A)
		p2 := ecs.Get[Position](w, evt.B)

		// Safety check if entities were destroyed mid-frame
		if p1 == nil || p2 == nil {
			continue
		}

		// Fetch mass components (default to standard mass if missing)
		mass1 := getMass(w, evt.A) // e.g., 0.73 for a 73 kg mage, 0 for static wall
		mass2 := getMass(w, evt.B)

		// Calculate inverse masses (1 / mass).
		// If mass is 0 (static wall), inverse mass is 0.
		invMass1 := float64(0)
		if mass1 > 0 {
			invMass1 = 1.0 / mass1
		}

		invMass2 := float64(0)
		if mass2 > 0 {
			invMass2 = 1.0 / mass2
		}

		totalInvMass := invMass1 + invMass2
		if totalInvMass == 0 {
			continue // Both are immovable objects (two walls)
		}

		// Apply positional correction proportional to inverse mass
		// Entity A moves away from B along the normal
		if invMass1 > 0 {
			ratio := invMass1 / totalInvMass
			p1.Center.X += evt.Normal.X * evt.Penetration * ratio
			p1.Center.Y += evt.Normal.Y * evt.Penetration * ratio
		}

		// Entity B moves in the opposite direction
		if invMass2 > 0 {
			ratio := invMass2 / totalInvMass
			p2.Center.X -= evt.Normal.X * evt.Penetration * ratio
			p2.Center.Y -= evt.Normal.Y * evt.Penetration * ratio
		}
	}
	// Truncate events without releasing memory
	s.events = s.events[:0]
}
