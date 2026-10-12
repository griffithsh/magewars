package game

import (
	"math"
	"time"

	"github.com/griffithsh/magewars/ecs"
)

const minValidLength = 1e-3

type Vector3 struct {
	X, Y, Z float64
}

type Velocity Vector3

func (v *Velocity) LengthSquared() float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}
func (v *Velocity) Length() float64 {
	return math.Sqrt(v.LengthSquared())
}

type Acceleration Vector3
type Force Vector3

func (f *Force) LengthSquared() float64 {
	return f.X*f.X + f.Y*f.Y + f.Z*f.Z
}

func (f *Force) Length() float64 {
	return math.Sqrt(f.LengthSquared())
}

func (f *Force) Clamp(max float64) {
	maxSq := max * max
	if f.LengthSquared() > maxSq {
		currentLength := math.Sqrt(f.LengthSquared())
		scale := max / currentLength

		f.X *= scale
		f.Y *= scale
		f.Z *= scale
	}
}

type Mass struct {
	Value float64 // Value is in hundredths of a kg
}

type MotionSystem struct {
}

func NewMotionSystem() *MotionSystem {
	return &MotionSystem{}
}

func (s *MotionSystem) Update(w *ecs.World, t time.Duration) error {
	entities := w.Query(ecs.Ty[Velocity](), ecs.Ty[Acceleration](), ecs.Ty[Force](), ecs.Ty[Mass](), ecs.Ty[Position]())

	dt := t.Seconds()
	if dt <= 0 || dt > 0.1 {
		dt = 1 / 60.0
	}

	for _, e := range entities {
		v, a, f, m, p := ecs.Get[Velocity](w, e), ecs.Get[Acceleration](w, e), ecs.Get[Force](w, e), ecs.Get[Mass](w, e), ecs.Get[Position](w, e)

		// FIXME: If an entity has no mass, accelerating it becomes weird,
		// right? any force becomes infinity speed? For now we just ignore
		// massless objects, but maybe this approach needs to be reexamined
		// later.
		if m.Value > 0 {
			// Apply Force and Mass to Acceleration
			a.X = f.X / m.Value
			a.Y = f.Y / m.Value
			a.Z = f.Z / m.Value
		} else {
			a.X = 0
			a.Y = 0
			a.Z = 0
		}

		// Apply Acceleration to Velocity
		v.X += a.X * dt
		v.Y += a.Y * dt
		v.Z += a.Z * dt

		// Apply drag to Velocity
		if gf := ecs.Get[GroundFriction](w, e); gf != nil {
			dragMultiplier := 1.0 - (gf.Drag * dt)
			if dragMultiplier < 0 {
				dragMultiplier = 0
			}
			v.X *= dragMultiplier
			v.Y *= dragMultiplier
		}

		// Cull trivially small Velocity values
		if v.LengthSquared() < minValidLength {
			v.X = 0
			v.Y = 0
			v.Z = 0
		} else {
			// Otherwise apply Velocity to Position
			p.Center.X += v.X * dt
			p.Center.Y += v.Y * dt
			// FIXME: Position does not capture an appropriate Z value with more
			// precision than integral "layer"s or a z index.
		}

		// Reset Acceleration and Force
		a.X = 0
		a.Y = 0
		a.Z = 0
		f.X = 0
		f.Y = 0
		f.Z = 0
	}
	return nil
}
