package game

import (
	"time"

	"github.com/griffithsh/magewars/ecs"
)

type MovementStats struct {
	MaxSpeed           float64
	MaxForce           float64
	BaseResponsiveness float64
}

type MovementIntent struct {
	// X and Y define desired velocity. This one is not normalised
	X, Y float64
}

type GroundFriction struct {
	Drag     float64 // Drag example values: Grass: 1.0, Mud: 2.0
	Traction float64 // Traction example values: Grass: 1.0, Ice: 0.1
}

type SteeringSystem struct {
}

func NewSteeringSystem() *SteeringSystem {
	return &SteeringSystem{}
}

func (s *SteeringSystem) Update(w *ecs.World, t time.Duration) error {
	entities := w.Query(ecs.Ty[MovementIntent](), ecs.Ty[GroundFriction](), ecs.Ty[Mass](), ecs.Ty[Force]())

	// deadStop is the kill zone for leftover velocity. If we get this low, then
	// just shed all movement to avoid weird low speed slides.
	const deadStop = 5.0 // 25.0 was previously recommended.

	// FIXME: maxForce and baseResponsiveness should be pulled from a
	// MovementStats Component that defines it per entity.
	const maxForce = 2000.0
	const baseResponsiveness = 0.135 // 0.16 is about right?
	const maxSpeed = 150.0

	for _, e := range entities {
		mi, v, gf, m, f := ecs.Get[MovementIntent](w, e), ecs.Get[Velocity](w, e), ecs.Get[GroundFriction](w, e), ecs.Get[Mass](w, e), ecs.Get[Force](w, e)
		// Calculate the DesiredVelocity from the MovementIntent and MaxSpeed
		dv := Vector3{
			X: mi.X * maxSpeed,
			Y: mi.Y * maxSpeed,
		}

		responsiveness := baseResponsiveness
		if dv.X == 0 && dv.Y == 0 {
			responsiveness = baseResponsiveness / 2.8 // 0.06
			if v.LengthSquared() < deadStop {
				v.X = 0
				v.Y = 0
				f.X = 0
				f.Y = 0
				continue
			}
		} else if (v.X*dv.X)+(v.Y*dv.Y) < 0 {
			responsiveness = baseResponsiveness / 2 // 0.08
		}
		tau := (responsiveness / gf.Traction)

		// Formula breakdown:
		// Force = Mass * (DesiredVelocity - Velocity) / (responsiveness / GroundFriction.Traction)
		f.X = m.Value * (dv.X - v.X) / tau
		f.Y = m.Value * (dv.Y - v.Y) / tau

		f.Clamp(maxForce)
	}

	return nil
}
