package game

import (
	"math"
	"testing"
	"time"

	"github.com/griffithsh/magewars/ecs"
	"github.com/griffithsh/magewars/event"
)

func TestCirclePolyCollisionCheck_FaceAndCorner(t *testing.T) {
	// A 64x64 square polygon centered around (100, 100): [68, 132] x [68, 132]
	// Vertices in clockwise order:
	polyVerts := []Vector2{
		{X: 68, Y: 68},   // top-left
		{X: 68, Y: 132},  // bottom-left
		{X: 132, Y: 132}, // bottom-right
		{X: 132, Y: 68},  // top-right
	}
	radius := 16.0

	t.Run("Approaching top face", func(t *testing.T) {
		// Circle center at (100, 58): overlaps top edge (Y=68) by 6 pixels
		circleCenter := Vector2{X: 100, Y: 58}
		collides, normal, pen := circlePolyCollisionCheck(circleCenter, radius, polyVerts)
		if !collides {
			t.Fatalf("expected collision")
		}
		if math.Abs(pen-6.0) > 1e-4 {
			t.Errorf("expected penetration 6.0, got %f", pen)
		}
		// Normal should point from polygon outward to circle (UP: X=0, Y=-1)
		if math.Abs(normal.X) > 1e-4 || math.Abs(normal.Y-(-1.0)) > 1e-4 {
			t.Errorf("expected normal (0, -1), got (%f, %f)", normal.X, normal.Y)
		}
	})

	t.Run("Approaching bottom face", func(t *testing.T) {
		// Circle center at (100, 142): overlaps bottom edge (Y=132) by 6 pixels
		circleCenter := Vector2{X: 100, Y: 142}
		collides, normal, pen := circlePolyCollisionCheck(circleCenter, radius, polyVerts)
		if !collides {
			t.Fatalf("expected collision")
		}
		if math.Abs(pen-6.0) > 1e-4 {
			t.Errorf("expected penetration 6.0, got %f", pen)
		}
		// Normal should point from polygon outward to circle (DOWN: X=0, Y=1)
		if math.Abs(normal.X) > 1e-4 || math.Abs(normal.Y-1.0) > 1e-4 {
			t.Errorf("expected normal (0, 1), got (%f, %f)", normal.X, normal.Y)
		}
	})

	t.Run("Approaching top-left corner", func(t *testing.T) {
		// Corner is at (68, 68). Circle at (68 - 6, 68 - 8) = (62, 60)
		// Distance to corner: sqrt(36 + 64) = 10
		// Radius 16 -> penetration = 6
		circleCenter := Vector2{X: 62, Y: 60}
		collides, normal, pen := circlePolyCollisionCheck(circleCenter, radius, polyVerts)
		if !collides {
			t.Fatalf("expected collision")
		}
		if math.Abs(pen-6.0) > 1e-4 {
			t.Errorf("expected penetration 6.0, got %f", pen)
		}
		// Normal points from (68, 68) to (62, 60): (-6/10, -8/10) = (-0.6, -0.8)
		if math.Abs(normal.X-(-0.6)) > 1e-4 || math.Abs(normal.Y-(-0.8)) > 1e-4 {
			t.Errorf("expected normal (-0.6, -0.8), got (%f, %f)", normal.X, normal.Y)
		}
	})

	t.Run("No collision when outside radius", func(t *testing.T) {
		circleCenter := Vector2{X: 100, Y: 40}
		collides, _, _ := circlePolyCollisionCheck(circleCenter, radius, polyVerts)
		if collides {
			t.Errorf("expected no collision")
		}
	})
}

func TestGapCollisionResolution(t *testing.T) {
	// Recreate the gap from the user's screenshot:
	// Tile 1 on left, Tile 2 on right.
	// Player circle (radius 16, diameter 32) trying to squeeze through 17px gap.
	bus := event.NewBus()
	colSys := NewCollisionSystem(bus)
	revSys := NewCollisionReversionSystem(bus)

	w := ecs.NewWorld()

	// Tile 1: polygon with right corner at (247, 138)
	t1 := w.CreateEntity()
	ecs.Add(w, t1, &Position{Center: Center{X: 224, Y: 160}})
	ecs.Add(w, t1, NewCollidablePolygon([]Vector2{
		{X: -24, Y: -20},
		{X: -22, Y: 25},
		{X: 25, Y: 26},
		{X: 23, Y: -22}, // world: (224+23, 160-22) = (247, 138)
	}))

	// Tile 2: polygon with left corner at (264, 140)
	t2 := w.CreateEntity()
	ecs.Add(w, t2, &Position{Center: Center{X: 288, Y: 160}})
	ecs.Add(w, t2, NewCollidablePolygon([]Vector2{
		{X: -24, Y: -20}, // world: (288-24, 160-20) = (264, 140)
		{X: -22, Y: 25},
		{X: 25, Y: 26},
		{X: 23, Y: -22},
	}))

	// Player circle at center of gap: X = 255.5, Y = 130
	player := w.CreateEntity()
	pPos := &Position{Center: Center{X: 255.5, Y: 130}}
	pVel := &Velocity{X: 0, Y: 100} // moving down into gap
	pMass := &Mass{Value: 0.89}
	pCirc := &CollidableCircle{Radius: 16}

	ecs.Add(w, player, pPos)
	ecs.Add(w, player, pVel)
	ecs.Add(w, player, pMass)
	ecs.Add(w, player, pCirc)

	dt := 16 * time.Millisecond

	// Update collision system
	if err := colSys.Update(w, dt); err != nil {
		t.Fatalf("colSys: %v", err)
	}

	if len(revSys.events) != 2 {
		t.Fatalf("expected 2 collision events, got %d", len(revSys.events))
	}

	// Update reversion system
	if err := revSys.Update(w, dt); err != nil {
		t.Fatalf("revSys: %v", err)
	}

	// Verify player was pushed UP (away from gap), not DOWN or sideways into the tiles
	if pPos.Center.Y >= 130 {
		t.Errorf("player should have been pushed UP (Y < 130), got Y = %f", pPos.Center.Y)
	}

	// Verify downward velocity was cancelled
	if pVel.Y > 0 {
		t.Errorf("downward velocity should be <= 0, got %f", pVel.Y)
	}
}

func TestMultiFramePushIntoGap(t *testing.T) {
	// Full simulation running Steering, Motion, Collision, CollisionReversion
	// with the player continuously pressing Down into the narrow gap.
	bus := event.NewBus()
	steeringSys := NewSteeringSystem()
	motionSys := NewMotionSystem()
	colSys := NewCollisionSystem(bus)
	revSys := NewCollisionReversionSystem(bus)

	w := ecs.NewWorld()

	// Tile 1: polygon with right corner at (247, 138)
	t1 := w.CreateEntity()
	ecs.Add(w, t1, &Position{Center: Center{X: 224, Y: 160}})
	ecs.Add(w, t1, NewCollidablePolygon([]Vector2{
		{X: -24, Y: -20},
		{X: -22, Y: 25},
		{X: 25, Y: 26},
		{X: 23, Y: -22}, // world: (247, 138)
	}))

	// Tile 2: polygon with left corner at (264, 140)
	t2 := w.CreateEntity()
	ecs.Add(w, t2, &Position{Center: Center{X: 288, Y: 160}})
	ecs.Add(w, t2, NewCollidablePolygon([]Vector2{
		{X: -24, Y: -20}, // world: (264, 140)
		{X: -22, Y: 25},
		{X: 25, Y: 26},
		{X: 23, Y: -22},
	}))

	player := w.CreateEntity()
	pPos := &Position{Center: Center{X: 255.5, Y: 100}}
	pVel := &Velocity{X: 0, Y: 0}
	pMass := &Mass{Value: 0.89}
	pCirc := &CollidableCircle{Radius: 16}
	pStats := &MovementStats{
		MaxSpeed:           111.0,
		MaxForce:           2000.0,
		BaseResponsiveness: 0.146,
	}
	pFriction := &GroundFriction{
		Drag:     1.0,
		Traction: 1.0,
	}
	pForce := &Force{}
	pAccel := &Acceleration{}

	ecs.Add(w, player, pPos)
	ecs.Add(w, player, pVel)
	ecs.Add(w, player, pMass)
	ecs.Add(w, player, pCirc)
	ecs.Add(w, player, pStats)
	ecs.Add(w, player, pFriction)
	ecs.Add(w, player, pForce)
	ecs.Add(w, player, pAccel)

	dt := 16 * time.Millisecond

	var lastY float64
	for frame := 0; frame < 180; frame++ { // 3 seconds of pushing down
		// Player constantly pushing DOWN
		ecs.Add(w, player, &MovementIntent{X: 0, Y: 1.0})

		if err := steeringSys.Update(w, dt); err != nil {
			t.Fatalf("steeringSys: %v", err)
		}
		if err := motionSys.Update(w, dt); err != nil {
			t.Fatalf("motionSys: %v", err)
		}
		if err := colSys.Update(w, dt); err != nil {
			t.Fatalf("colSys: %v", err)
		}
		if err := revSys.Update(w, dt); err != nil {
			t.Fatalf("revSys: %v", err)
		}

		// The player must NEVER squeeze through into the gap (Y must stay < 135)
		if pPos.Center.Y > 135 {
			t.Fatalf("frame %d: player squeezed through! Y = %f", frame, pPos.Center.Y)
		}
		// The player must NEVER be ejected to the right into Tile 2 (X must stay near 255.5)
		if math.Abs(pPos.Center.X-255.5) > 10 {
			t.Fatalf("frame %d: player ejected sideways! X = %f", frame, pPos.Center.X)
		}
		// In the last 60 frames (steady state on boundary), check that jitter is under 0.5 pixels
		if frame > 120 {
			jitter := math.Abs(pPos.Center.Y - lastY)
			if jitter > 0.5 {
				t.Fatalf("frame %d: excessive jitter between frames: %f px", frame, jitter)
			}
		}
		lastY = pPos.Center.Y
	}
}

func TestWallCollision_Sliding(t *testing.T) {
	// Verifies that a player moving diagonally into a flat obstacle slides
	// cleanly along the boundary without penetration or micro-jitter.
	bus := event.NewBus()
	steeringSys := NewSteeringSystem()
	motionSys := NewMotionSystem()
	colSys := NewCollisionSystem(bus)
	revSys := NewCollisionReversionSystem(bus)

	w := ecs.NewWorld()

	wall := w.CreateEntity()
	ecs.Add(w, wall, &Position{Center: Center{X: 100, Y: 132}})
	ecs.Add(w, wall, NewCollidablePolygon([]Vector2{
		{X: -64, Y: -32}, // Y=100
		{X: -64, Y: 32},
		{X: 64, Y: 32},
		{X: 64, Y: -32},
	}))

	player := w.CreateEntity()
	pPos := &Position{Center: Center{X: 100, Y: 84}} // starting right on boundary (wall Y=100, radius=16)
	pVel := &Velocity{X: 0, Y: 0}
	pMass := &Mass{Value: 0.89}
	pCirc := &CollidableCircle{Radius: 16}
	pStats := &MovementStats{
		MaxSpeed:           111.0,
		MaxForce:           2000.0,
		BaseResponsiveness: 0.146,
	}
	pFriction := &GroundFriction{
		Drag:     1.0,
		Traction: 1.0,
	}
	pForce := &Force{}
	pAccel := &Acceleration{}

	ecs.Add(w, player, pPos)
	ecs.Add(w, player, pVel)
	ecs.Add(w, player, pMass)
	ecs.Add(w, player, pCirc)
	ecs.Add(w, player, pStats)
	ecs.Add(w, player, pFriction)
	ecs.Add(w, player, pForce)
	ecs.Add(w, player, pAccel)

	dt := 16 * time.Millisecond

	for frame := 0; frame < 30; frame++ {
		// Player pressing Down-Right diagonally
		ecs.Add(w, player, &MovementIntent{X: 1.0, Y: 1.0})
		if err := steeringSys.Update(w, dt); err != nil {
			t.Fatalf("steeringSys: %v", err)
		}
		if err := motionSys.Update(w, dt); err != nil {
			t.Fatalf("motionSys: %v", err)
		}
		if err := colSys.Update(w, dt); err != nil {
			t.Fatalf("colSys: %v", err)
		}
		if err := revSys.Update(w, dt); err != nil {
			t.Fatalf("revSys: %v", err)
		}

		// Vertical boundary is strictly maintained at Y=84
		if math.Abs(pPos.Center.Y-84.0) > 1e-4 {
			t.Errorf("frame %d: expected Y=84.0, got %f", frame, pPos.Center.Y)
		}
		// Normal velocity into the wall is cancelled
		if math.Abs(pVel.Y) > 1e-4 {
			t.Errorf("frame %d: expected Vel.Y=0, got %f", frame, pVel.Y)
		}
	}

	// Player should have smoothly slid rightward along the wall
	if pPos.Center.X <= 120 {
		t.Errorf("expected player to slide rightward (X > 120), got X = %f", pPos.Center.X)
	}
	if pVel.X <= 50 {
		t.Errorf("expected player to maintain tangential velocity (Vel.X > 50), got Vel.X = %f", pVel.X)
	}
}




