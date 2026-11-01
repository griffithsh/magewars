package data

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"image"
	"image/png"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/griffithsh/magewars/embedded"
	"github.com/griffithsh/magewars/internal/game"
)

// Archive is a store of game data.
type Archive struct {
	images        map[string]image.Image
	animations    map[string][]game.Frame // FIXME: why not a pointer here?
	levels        map[string]*game.Level
	tiledMaps     map[string]*tiledMap
	tiledTilesets map[string]*tiledTileset
}

// NewArchive constructs a new Archive.
func NewArchive() (*Archive, error) {
	archive := Archive{
		images:        map[string]image.Image{},
		animations:    map[string][]game.Frame{},
		levels:        map[string]*game.Level{},
		tiledMaps:     map[string]*tiledMap{},
		tiledTilesets: map[string]*tiledTileset{},
	}
	// Load all the embedded contents in here.
	files, err := embedded.Filenames()
	if err != nil {
		return nil, fmt.Errorf("list embedded files: %v", err)
	}
	for _, filename := range files {
		b, err := embedded.Get(filename)
		if err != nil {
			return nil, fmt.Errorf("get file %q: %v", filename, err)
		}
		if err := archive.interpret(filename, bytes.NewReader(b)); err != nil {
			return nil, fmt.Errorf("interpret %q: %v", filename, err)
		}
	}

	return &archive, nil
}

// Load data into the Archive from a tar.gz archive, replacing data already in
// the Archive if the provided files share the same filenames.
func (a *Archive) Load(r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("new reader: %v", err)
	}

	tr := tar.NewReader(gzr)

	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("read next file from tar: %v", err)
		}

		if head.Typeflag == tar.TypeReg {
			if err := a.interpret(head.Name, tr); err != nil {
				return fmt.Errorf("interpret %q: %v", head.Name, err)
			}
		}
	}

	return nil
}

func (a *Archive) interpret(filename string, r io.Reader) error {
	// getExt determines the file extension with special rules - it includes the
	// second level of extension too. Inputting sample.foo.txt would return
	// ".foo.txt". Whereas sample.txt returns just ".txt". A .json file just
	// declares the encoding, but a *.world.json file hints at a schema also.
	getExt := func(filename string) string {
		separated := strings.Split(filename, ".")
		encoding := separated[len(separated)-1]
		if len(separated) > 2 {
			return fmt.Sprintf(".%s.%s", separated[len(separated)-2], encoding)
		}
		return "." + encoding
	}

	switch getExt(filename) {
	case ".png":
		decoded, err := png.Decode(r)
		if err != nil {
			return fmt.Errorf("png.Decode %s: %s", filename, err)
		}
		a.images[filename] = decoded
	case ".anim.csv":
		// decode!
		raw, err := csv.NewReader(r).ReadAll()
		if err != nil {
			return fmt.Errorf("csv reader.ReadAll %s: %s", filename, err)
		}
		frames := []game.Frame{}
		for i, row := range raw {
			frame := game.Frame{
				Texture: row[0],
			}
			frame.X, err = strconv.Atoi(row[1])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 1", err, filename, i)
			}
			frame.Y, err = strconv.Atoi(row[2])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 2", err, filename, i)
			}
			frame.W, err = strconv.Atoi(row[3])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 3", err, filename, i)
			}
			frame.H, err = strconv.Atoi(row[4])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 4", err, filename, i)
			}
			frame.XOffset, err = strconv.Atoi(row[5])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 5", err, filename, i)
			}
			frame.YOffset, err = strconv.Atoi(row[6])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 6", err, filename, i)
			}
			duration, err := strconv.Atoi(row[7])
			if err != nil {
				return fmt.Errorf("syntax error (strconv.Atoi: %s): %s, row %d, column 6", err, filename, i)
			}
			frame.Duration = time.Duration(duration) * time.Millisecond

			frames = append(frames, frame)
		}
		a.animations[filename] = frames
	case ".tmx":
		tmx, err := interpretTMX(r)
		if err != nil {
			return err
		}
		a.tiledMaps[filename] = tmx
	case ".tsx":
		tsx, err := interpretTSX(r)
		if err != nil {
			return err
		}
		a.tiledTilesets[filename] = tsx
	case ".gitkeep":
		// ignored
	default:
		fmt.Fprintf(os.Stderr, "WARNING: no resource loader handled %q, detected extension was %q\n", filename, getExt(filename))
	}

	return nil
}

// GetImage returns an image that has been loaded into the archive.
func (a *Archive) GetImage(name string) (val image.Image, ok bool) {
	val, ok = a.images[name]
	return val, ok
}

// GetAnimation returns an animation by its file name.
func (a *Archive) GetAnimation(name string) (val []game.Frame, ok bool) {
	val, ok = a.animations[name]
	return val, ok
}

func getGIDsFromLayer(layer *tiledLayer) [][]game.GID {
	result := [][]game.GID{}
	rows := strings.Split(strings.ReplaceAll(strings.TrimSpace(layer.Data.Content), "\r\n", "\n"), "\n")
	for _, row := range rows {
		rawGIDs := strings.Split(strings.TrimRight(row, ","), ",")
		row := []game.GID{}
		for _, rawGID := range rawGIDs {
			val, err := strconv.Atoi(rawGID)
			if err != nil {
				continue
			}
			row = append(row, game.GID(val))
		}
		result = append(result, row)
	}
	return result
}

func getUniqueGIDsFromContent(m *tiledMap) map[game.GID]struct{} {
	result := make(map[game.GID]struct{})
	for _, layer := range m.Layers {
		rows := strings.Split(strings.ReplaceAll(strings.TrimSpace(layer.Data.Content), "\r\n", "\n"), "\n")
		for _, row := range rows {
			rawGIDs := strings.Split(strings.TrimRight(row, ","), ",")
			for _, rawGID := range rawGIDs {
				val, err := strconv.Atoi(rawGID)
				if err != nil || val == 0 {
					continue
				}
				result[game.GID(val)] = struct{}{}
			}
		}
	}
	return result
}

func getTiles(gids []game.GID, tilesets map[string]*tiledTileset) (map[game.GID]game.Tile, error) {
	// FIXME: This grabs any tileset and hopes for the best. This will only work
	// while there is a single tileset in the map. Fixing this would require
	// firstGID counting and offsetting.
	var tsx *tiledTileset
	for _, v := range tilesets {
		tsx = v
	}

	result := make(map[game.GID]game.Tile)

	// here gids are the 1-based gids from a map

	for _, gid := range gids {
		srcX := int(int(gid-1)%(tsx.Image.Width/tsx.TileWidth)) * tsx.TileWidth
		srcY := int(int(gid-1)/(tsx.Image.Height/tsx.TileHeight)) * tsx.TileHeight
		fmt.Printf("for tile gid %d, clip from (%d,%d)\n", gid, srcX, srcY)

		t := game.Tile{
			Texture:   tsx.Image.Source,
			X:         srcX,
			Y:         srcY,
			W:         tsx.TileWidth,
			H:         tsx.TileHeight,
			Collision: []game.Vector2{},
		}

		// findMatchingTilesetTileInfo

		// If this GID is reffed by an object tile thingy ...?
		for _, tilesetTile := range tsx.Tiles {
			if game.GID(tilesetTile.ID+1) != gid {
				continue
			}
			pairs := strings.Split(tilesetTile.ObjectGroup.Objects[0].Polygon.Points, " ")
			if len(pairs) == 0 {
				continue
			}
			for _, pair := range pairs {
				if pair == "" {
					continue
				}
				rawCoords := strings.Split(pair, ",")

				x, err := strconv.ParseFloat(rawCoords[0], 64)
				if err != nil {
					return nil, fmt.Errorf("bad raw coords[0]: %q\n", rawCoords[0])
				}
				y, err := strconv.ParseFloat(rawCoords[1], 64)
				if err != nil {
					return nil, fmt.Errorf("bad raw coords[1]: %q\n", rawCoords[1])
				}
				t.Collision = append(t.Collision, game.Vector2{
					// NB: Using the dimensions of the tile here, which *can* be
					// different to the space reserved for a tile in a map's
					// grid. This may be the source of a bug in future.
					X: x - float64(tsx.TileWidth)/2,
					Y: y - float64(tsx.TileHeight)/2,
				})

			}
			break
		}
		result[gid] = t
	}
	return result, nil
}

func (a *Archive) GetLevel(name string) (val *game.Level, ok bool) {
	val, ok = a.levels[name]
	if ok {
		return val, ok
	}

	// We didn't find this Level already, so compose a new Level object from a
	// tmx and the one or more tsx files that it references.
	m, ok := a.tiledMaps[name+".tmx"]
	if !ok {
		fmt.Printf("Not finding %q in %v", name, a.tiledMaps)
		return nil, false
	}

	// FIXME: only one tileset is supported with this construct! Multiple tilesets means implementing GID offsetting.
	if len(m.Tilesets) != 1 {
		panic("Cannot handle a .tmx with multiple tilesets right now!")
	}
	uniqueGIDs := getUniqueGIDsFromContent(m)

	// FIXME: tilesets here is a cache of Tiles, reused across the Level, but
	// individual Tiles can be loaded multiple times if they are used across
	// multiple levels. This is a waste of memory if used at large scale.
	tilesets, err := getTiles(slices.Collect(maps.Keys(uniqueGIDs)), a.tiledTilesets)
	if err != nil {
		fmt.Printf("couldn't get the tilesets\n")
		return nil, false
	}

	// m.Tilesets[0].FirstGid

	fmt.Printf("full of tilesets: %v\n", tilesets)

	tiles := []game.PositionedTile{}
	for _, layer := range m.Layers {
		gids := getGIDsFromLayer(&layer)
		for wy, row := range gids {
			for wx, gid := range row {
				if gid == 0 {
					// NB Making the assumption that a gid of zero is a Tiled
					// semaphore for "nothing here".
					continue
				}
				tile, ok := tilesets[gid]
				if !ok {
					panic(fmt.Sprintf("missing a tile for gid: %v", gid))
				}
				pTile := game.PositionedTile{
					X:   float64(wx*m.TileWidth + m.TileWidth/2),
					Y:   float64(wy*m.TileHeight + m.TileHeight/2),
					Ref: &tile,
				}
				tiles = append(tiles, pTile)
			}
		}
	}

	level := game.Level{
		Name:       name,
		TileWidth:  m.TileWidth,
		TileHeight: m.TileHeight,
		Tiles:      tiles,
		Tileset:    tilesets,
	}

	a.levels[name] = &level

	// Redo the call to find what we just cached.
	return a.GetLevel(name)
}
