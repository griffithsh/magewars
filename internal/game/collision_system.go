package game

import (
	"math"
	"time"

	"github.com/griffithsh/magewars/ecs"
	"github.com/griffithsh/magewars/event"
)

type Vector2 struct {
	X, Y float64
}

type CollisionEvent struct {
	A, B        ecs.Entity
	Normal      Vector2 // Direction to push Entity A (points from B to A)
	Penetration float64 // How far they are overlapping (in pixels)
}

func (v *Vector2) LengthSquared() float64 {
	return v.X*v.X + v.Y*v.Y
}

type CollidablePolygon struct {
	Vertices []Vector2
	normals  []Vector2
	// BoundingRadius float64 // TODO: potential optimisation here
}

// PolygonSignedArea computes the signed area of a 2D polygon using the Shoelace formula.
// In screen coordinates (+X right, +Y down):
// - Negative signed area indicates Counter-Clockwise (CCW) winding.
// - Positive signed area indicates Clockwise (CW) winding.
func PolygonSignedArea(vertices []Vector2) float64 {
	n := len(vertices)
	if n < 3 {
		return 0
	}
	area := 0.0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += vertices[i].X*vertices[j].Y - vertices[j].X*vertices[i].Y
	}
	return area / 2.0
}

func NewCollidablePolygon(vertices []Vector2) *CollidablePolygon {
	// Ensure vertices are in counter-clockwise order for consistent outward normals.
	verts := make([]Vector2, len(vertices))
	copy(verts, vertices)
	if PolygonSignedArea(verts) > 0 {
		// Clockwise: reverse to make counter-clockwise
		for i, j := 0, len(verts)-1; i < j; i, j = i+1, j-1 {
			verts[i], verts[j] = verts[j], verts[i]
		}
	}

	return &CollidablePolygon{
		Vertices: verts,
		normals:  computeOutwardNormals(verts),
	}
}

type CollidableCircle struct {
	Radius float64
}

type CollisionSystem struct {
	bus *event.Bus
}

func NewCollisionSystem(bus *event.Bus) *CollisionSystem {
	return &CollisionSystem{bus: bus}
}

func computeOutwardNormals(vertices []Vector2) []Vector2 {
	n := len(vertices)
	normals := make([]Vector2, n)

	for i := 0; i < n; i++ {
		// Get the current vertex and the next one (wrapping around at the end)
		p1 := vertices[i]
		p2 := vertices[(i+1)%n]

		// 1. Calculate the edge vector (direction from p1 to p2)
		edge := Vector2{
			X: p2.X - p1.X,
			Y: p2.Y - p1.Y,
		}

		// 2. In screen coordinates (+Y down), for a counter-clockwise polygon,
		// the outward-facing normal is (-edge.Y, edge.X).
		normal := Vector2{
			X: -edge.Y,
			Y: edge.X,
		}

		// 3. Normalize it (make its length = 1)
		length := math.Sqrt(float64(normal.X*normal.X + normal.Y*normal.Y))
		if length > 0 {
			normal.X /= length
			normal.Y /= length
		}

		normals[i] = normal
	}

	return normals
}

// circlePolyCollisionCheck returns whether a circle and a polygon intersect,
// and if they do, the normal pointing outward from the polygon toward the circle,
// and the penetration depth.
func circlePolyCollisionCheck(circleCenter Vector2, circleRadius float64, polyVerts []Vector2) (bool, Vector2, float64) {
	n := len(polyVerts)
	if n < 3 {
		return false, Vector2{}, 0
	}

	minDistSq := math.MaxFloat64
	var closestPoint Vector2
	var closestEdgeDir Vector2

	for i := 0; i < n; i++ {
		p1 := polyVerts[i]
		p2 := polyVerts[(i+1)%n]

		edge := Vector2{X: p2.X - p1.X, Y: p2.Y - p1.Y}
		eLenSq := edge.X*edge.X + edge.Y*edge.Y
		if eLenSq == 0 {
			continue
		}

		v := Vector2{X: circleCenter.X - p1.X, Y: circleCenter.Y - p1.Y}
		t := (v.X*edge.X + v.Y*edge.Y) / eLenSq
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}

		pt := Vector2{X: p1.X + t*edge.X, Y: p1.Y + t*edge.Y}
		dx := circleCenter.X - pt.X
		dy := circleCenter.Y - pt.Y
		distSq := dx*dx + dy*dy

		if distSq < minDistSq {
			minDistSq = distSq
			closestPoint = pt
			closestEdgeDir = edge
		}
	}

	// Ray casting to determine if circleCenter is inside the polygon
	inside := false
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		vi := polyVerts[i]
		vj := polyVerts[j]
		if (vi.Y > circleCenter.Y) != (vj.Y > circleCenter.Y) &&
			circleCenter.X < (vj.X-vi.X)*(circleCenter.Y-vi.Y)/(vj.Y-vi.Y)+vi.X {
			inside = !inside
		}
	}

	dist := math.Sqrt(minDistSq)

	if !inside {
		if dist >= circleRadius {
			return false, Vector2{}, 0
		}
		penetration := circleRadius - dist
		var normal Vector2
		if dist > 1e-9 {
			normal = Vector2{
				X: (circleCenter.X - closestPoint.X) / dist,
				Y: (circleCenter.Y - closestPoint.Y) / dist,
			}
		} else {
			eLen := math.Sqrt(closestEdgeDir.X*closestEdgeDir.X + closestEdgeDir.Y*closestEdgeDir.Y)
			if eLen > 0 {
				normal = Vector2{X: closestEdgeDir.Y / eLen, Y: -closestEdgeDir.X / eLen}
			} else {
				normal = Vector2{X: 1, Y: 0}
			}
		}
		return true, normal, penetration
	}

	// circleCenter is inside polygon
	penetration := circleRadius + dist
	var normal Vector2
	if dist > 1e-9 {
		// Vector from circleCenter toward closest edge point (pushing outward)
		normal = Vector2{
			X: (closestPoint.X - circleCenter.X) / dist,
			Y: (closestPoint.Y - circleCenter.Y) / dist,
		}
	} else {
		eLen := math.Sqrt(closestEdgeDir.X*closestEdgeDir.X + closestEdgeDir.Y*closestEdgeDir.Y)
		if eLen > 0 {
			normal = Vector2{X: closestEdgeDir.Y / eLen, Y: -closestEdgeDir.X / eLen}
		} else {
			normal = Vector2{X: 1, Y: 0}
		}
	}
	return true, normal, penetration
}

// projectPolygon is a helper to project a polygon onto an axis and return the
// [min, max] interval.
func projectPolygon(axis Vector2, vertices []Vector2) (float64, float64) {
	// Initialize min and max with the dot product of the first vertex
	min := vertices[0].X*axis.X + vertices[0].Y*axis.Y
	max := min

	for i := 1; i < len(vertices); i++ {
		p := vertices[i].X*axis.X + vertices[i].Y*axis.Y
		if p < min {
			min = p
		} else if p > max {
			max = p
		}
	}
	return min, max
}

func overlapExists(minA, maxA, minB, maxB float64) bool {
	// They overlap if MinA is less than MaxB AND MinB is less than MaxA
	return minA <= maxB && minB <= maxA
}

// polyPolyCollisionCheck returns whether two polygons intersect, and if they
// do, the smallestAxis and penetration depth.
func polyPolyCollisionCheck(vertsA, normalsA, vertsB, normalsB []Vector2) (bool, Vector2, float64) {
	minOverlap := math.MaxFloat64
	var smallestAxis Vector2

	// 1. Test all normals of Polygon A
	for _, axis := range normalsA {
		minA, maxA := projectPolygon(axis, vertsA)
		minB, maxB := projectPolygon(axis, vertsB)

		if !overlapExists(minA, maxA, minB, maxB) {
			return false, Vector2{}, 0
		}

		overlap := math.Min(maxA, maxB) - math.Max(minA, minB)
		if overlap < minOverlap {
			minOverlap = overlap
			smallestAxis = axis
		}
	}

	// 2. Test all normals of Polygon B
	for _, axis := range normalsB {
		minA, maxA := projectPolygon(axis, vertsA)
		minB, maxB := projectPolygon(axis, vertsB)

		if !overlapExists(minA, maxA, minB, maxB) {
			return false, Vector2{}, 0
		}

		overlap := math.Min(maxA, maxB) - math.Max(minA, minB)
		if overlap < minOverlap {
			minOverlap = overlap
			smallestAxis = axis
		}
	}

	return true, smallestAxis, minOverlap
}

// circleCircleCollisionCheck returns whether two circles intersect, and if they
// do, the smallestAxis and penetration depth.
func circleCircleCollisionCheck(p1, p2 *Position, c1, c2 *CollidableCircle) (bool, Vector2, float64) {
	// Vector from Circle 2 to Circle 1
	dx := p1.Center.X - p2.Center.X
	dy := p1.Center.Y - p2.Center.Y
	distSq := (dx * dx) + (dy * dy)

	sumRadius := c1.Radius + c2.Radius

	// If squared distance is greater than or equal to squared sum of radii, no collision
	if distSq >= (sumRadius * sumRadius) {
		return false, Vector2{}, 0
	}

	dist := math.Sqrt(distSq)
	var normal Vector2

	if dist > 0 {
		// Normal points from Circle 2 to Circle 1
		normal = Vector2{X: dx / dist, Y: dy / dist}
	} else {
		// Edge case: If their centers are in the exact same spot,
		// default to pushing them along the X axis
		normal = Vector2{X: 1, Y: 0}
	}

	// Penetration depth is the difference between the sum of radii and the actual distance
	depth := sumRadius - dist

	return true, normal, depth
}

func localToWorld(x, y float64, vertices []Vector2) []Vector2 {
	world := make([]Vector2, len(vertices))
	for i, v := range vertices {
		world[i] = Vector2{X: x + v.X, Y: y + v.Y}
	}
	return world
}

func (s *CollisionSystem) Update(w *ecs.World, t time.Duration) error {
	entities := w.QueryAny(ecs.Ty[CollidablePolygon](), ecs.Ty[CollidableCircle]())
	for i1, e1 := range entities {
		p1 := ecs.Get[Position](w, e1)
		if p1 == nil {
			// Cannot collide if you dont have a position
			continue
		}
		v1 := ecs.Get[Velocity](w, e1)
		for i2, e2 := range entities {
			// We iterate against the same list twice. We don't need to scan the entire list on the inner loop.
			if i1 >= i2 {
				continue
			}

			// Cannot collide if you dont have a position
			p2 := ecs.Get[Position](w, e2)
			if p2 == nil {
				continue
			}

			// If neither are moving, is it safe to assume that there is no collision?
			v2 := ecs.Get[Velocity](w, e2)
			if v1 == nil && v2 == nil {
				continue
			}

			poly1, c1, poly2, c2 := ecs.Get[CollidablePolygon](w, e1), ecs.Get[CollidableCircle](w, e1), ecs.Get[CollidablePolygon](w, e2), ecs.Get[CollidableCircle](w, e2)

			// Have two circles collided?
			if c1 != nil && c2 != nil {
				if collides, normal, penetration := circleCircleCollisionCheck(p1, p2, c1, c2); collides {
					// Collision!
					event.Publish(s.bus, &CollisionEvent{A: e1, B: e2, Normal: normal, Penetration: penetration})
					continue
				}
			}

			// Have two polygons collided?
			if poly1 != nil && poly2 != nil {
				worldVerts1 := localToWorld(p1.Center.X, p1.Center.Y, poly1.Vertices)
				worldVerts2 := localToWorld(p2.Center.X, p2.Center.Y, poly2.Vertices)
				if collides, normal, penetration := polyPolyCollisionCheck(worldVerts1, poly1.normals, worldVerts2, poly2.normals); collides {
					// Collision!
					// Ensure normal points from B to A
					dirX := p1.Center.X - p2.Center.X
					dirY := p1.Center.Y - p2.Center.Y

					if (dirX*normal.X + dirY*normal.Y) < 0 {
						normal.X = -normal.X
						normal.Y = -normal.Y
					}

					event.Publish(s.bus, &CollisionEvent{
						A:           e1,
						B:           e2,
						Normal:      normal,
						Penetration: penetration,
					})
					continue
				}
			}

			// Has a circle and a polygon collided?
			// Note: circlePolyCollisionCheck returns normal pointing from Polygon to Circle.
			// CollisionEvent requires Normal pointing from B to A (direction to push Entity A).
			if poly1 != nil && c2 != nil {
				worldVerts := localToWorld(p1.Center.X, p1.Center.Y, poly1.Vertices)
				if collides, normal, penetration := circlePolyCollisionCheck(Vector2{X: p2.Center.X, Y: p2.Center.Y}, c2.Radius, worldVerts); collides {
					// Here A is poly1 and B is c2.
					// normal points from poly1 (A) to c2 (B).
					// Therefore normal pointing from B to A is -normal.
					event.Publish(s.bus, &CollisionEvent{
						A:           e1,
						B:           e2,
						Normal:      Vector2{X: -normal.X, Y: -normal.Y},
						Penetration: penetration,
					})
					continue
				}
			} else if c1 != nil && poly2 != nil {
				worldVerts := localToWorld(p2.Center.X, p2.Center.Y, poly2.Vertices)
				if collides, normal, penetration := circlePolyCollisionCheck(Vector2{X: p1.Center.X, Y: p1.Center.Y}, c1.Radius, worldVerts); collides {
					// Here A is c1 and B is poly2.
					// normal points from poly2 (B) to c1 (A).
					// Therefore normal is already pointing from B to A.
					event.Publish(s.bus, &CollisionEvent{
						A:           e1,
						B:           e2,
						Normal:      normal,
						Penetration: penetration,
					})
					continue
				}
			}
		}
	}

	return nil
}
