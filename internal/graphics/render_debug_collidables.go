package graphics

import (
	"math"

	"github.com/griffithsh/magewars/ecs"
	"github.com/griffithsh/magewars/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// renderDebugCollidables draws semitransparent shapes around collision
// Components, so that bugs in collision logic are easier to spot.
func (r *Renderer) renderDebugCollidables(w *ecs.World, screen *ebiten.Image, focusX, focusY, zoom float64) {
	screenW, screenH := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	fillOptions := vector.FillOptions{}
	var fillColorScale ebiten.ColorScale
	fillColorScale.Set(0.5, 0.5, 0.5, 0.57)
	drawFillOpts := vector.DrawPathOptions{
		ColorScale: fillColorScale,
		Blend:      ebiten.BlendSourceOver,
		AntiAlias:  true,
	}
	strokeOptions := vector.StrokeOptions{
		Width:    1.2,
		LineJoin: vector.LineJoinRound,
	}
	var strokeColorScale ebiten.ColorScale
	strokeColorScale.Set(1, 1, 1, 1)
	drawStrokeOpts := vector.DrawPathOptions{
		ColorScale: strokeColorScale,
		Blend:      ebiten.BlendSourceOver,
		AntiAlias:  true,
	}

	entities := w.QueryAny(ecs.Ty[game.CollidableCircle](), ecs.Ty[game.CollidablePolygon]())
	for _, e := range entities {
		p := ecs.Get[game.Position](w, e)
		if p == nil {
			continue
		}

		circle := ecs.Get[game.CollidableCircle](w, e)
		poly := ecs.Get[game.CollidablePolygon](w, e)

		var path vector.Path
		if circle != nil {
			path.Arc(
				float32(p.Center.X*zoom+screenW/2),
				float32(p.Center.Y*zoom+screenH/2),
				float32(circle.Radius*zoom),
				0,
				float32(2*math.Pi),
				vector.Clockwise,
			)
			path.Close()
		} else if poly != nil && len(poly.Vertices) > 0 {
			path.MoveTo(
				float32((poly.Vertices[0].X+p.Center.X)*zoom+screenW/2),
				float32((poly.Vertices[0].Y+p.Center.Y)*zoom+screenH/2),
			)
			for _, vert := range poly.Vertices[1:] {
				path.LineTo(
					float32((vert.X+p.Center.X)*zoom+screenW/2),
					float32((vert.Y+p.Center.Y)*zoom+screenH/2),
				)
			}
			path.Close()
		} else {
			continue
		}

		vector.FillPath(screen, &path, &fillOptions, &drawFillOpts)
		vector.StrokePath(screen, &path, &strokeOptions, &drawStrokeOpts)
	}
}
