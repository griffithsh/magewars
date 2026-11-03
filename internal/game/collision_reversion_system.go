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
		events:    make([]*CollisionEvent, 0, maxEvents/10),
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

func (s *CollisionReversionSystem) Update(w *ecs.World, t time.Duration) error {
	for _, evt := range s.events {
		p1 := ecs.Get[Position](w, evt.A)
		p2 := ecs.Get[Position](w, evt.B)

		if p1 == nil || p2 == nil {
			continue
		}

		mass1 := getMass(w, evt.A)
		mass2 := getMass(w, evt.B)

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
			continue
		}
		// 1. Positional correction: resolve penetration along collision normal.
		// evt.Normal points from B to A (separation direction for A).
		if evt.Penetration > 0 {
			correctionMagnitude := evt.Penetration

			if invMass1 > 0 {
				ratio := invMass1 / totalInvMass
				p1.Center.X += evt.Normal.X * correctionMagnitude * ratio
				p1.Center.Y += evt.Normal.Y * correctionMagnitude * ratio
			}
			if invMass2 > 0 {
				ratio := invMass2 / totalInvMass
				p2.Center.X -= evt.Normal.X * correctionMagnitude * ratio
				p2.Center.Y -= evt.Normal.Y * correctionMagnitude * ratio
			}
		}

		// 2. Velocity correction: cancel relative velocity moving into the collision normal
		// so the entity stops dead on the boundary and doesn't tunnel back in next frame.
		v1 := ecs.Get[Velocity](w, evt.A)
		v2 := ecs.Get[Velocity](w, evt.B)

		relVelX := float64(0)
		relVelY := float64(0)
		if v1 != nil {
			relVelX += v1.X
			relVelY += v1.Y
		}
		if v2 != nil {
			relVelX -= v2.X
			relVelY -= v2.Y
		}

		velAlongNormal := relVelX*evt.Normal.X + relVelY*evt.Normal.Y

		if velAlongNormal < 0 {
			restitution := 0.0 // inelastic: stop dead on boundary
			impulseScalar := -(1.0 + restitution) * velAlongNormal / totalInvMass

			if v1 != nil && invMass1 > 0 {
				v1.X += evt.Normal.X * impulseScalar * invMass1
				v1.Y += evt.Normal.Y * impulseScalar * invMass1
			}
			if v2 != nil && invMass2 > 0 {
				v2.X -= evt.Normal.X * impulseScalar * invMass2
				v2.Y -= evt.Normal.Y * impulseScalar * invMass2
			}
		}
	}

	s.events = s.events[:0]
	return nil
}
