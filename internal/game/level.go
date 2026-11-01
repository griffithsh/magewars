package game

import (
	"github.com/griffithsh/magewars/ecs"
)

type Tile struct {
	Texture string
	X, Y    int // Top-left texture coordinate
	W, H    int // width and height
	// Collision is optionally, a center-relative list of verts, anticlockwise
	// order, describing a polygon of impassable area of this Tile.
	Collision []Vector2
}

type PositionedTile struct {
	X, Y float64 // Center of tile in world coordinates
	Ref  *Tile   // Reference to a shared Tile definition so we don't load it into memory repeatedly.
}

type GID int

type Level struct {
	Name       string // might be handy to identify these at some point?
	TileWidth  int
	TileHeight int

	// Tiles is currently the full list of all Tiles in the Level, without any
	// layering information. Future usecases may require that we separate this
	// into conventional layers, like a layer for the base ground tiles, a layer
	// for shrubs, rocks, and other things that mages might need to be Y-sorted
	// against, and a layer of things that are always above everything, like
	// uper tree branches.
	Tiles []PositionedTile

	// Tileset is a flattened map of the recipes to create Tiles.
	Tileset map[GID]Tile
}

// LoadLevel takes a Level as a recipe, and creates Entities with Components
// that fully materialize it. It returns a root Entity that will cleanup the
// level when it is DestroyEntity()-ed.
func LoadLevel(level *Level, w *ecs.World) (ecs.Entity, error) {
	root := w.CreateEntity()

	// for each tile in the map, create an entity with a sprite and position
	// if the tile has collision info, also create a CollidablePolygon
	for _, tile := range level.Tiles {
		e := w.CreateEntity()
		w.Dependency(root, e)

		ecs.Add(w, e, &Position{
			Center: Center{
				X: tile.X,
				Y: tile.Y,
			},
			Layer:    1,
			Absolute: false,
		})
		ecs.Add(w, e, &Sprite{
			Texture: tile.Ref.Texture,
			X:       tile.Ref.X,
			Y:       tile.Ref.Y,
			W:       tile.Ref.W,
			H:       tile.Ref.H,
		})

		if len(tile.Ref.Collision) > 0 {
			ecs.Add(w, e, NewCollidablePolygon(tile.Ref.Collision))
		}
	}

	return root, nil
}
