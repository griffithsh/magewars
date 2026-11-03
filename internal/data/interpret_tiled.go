package data

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type tiledMap struct {
	XMLName    xml.Name                `xml:"map"`
	Version    string                  `xml:"version,attr"`
	TileWidth  int                     `xml:"tilewidth,attr"`
	TileHeight int                     `xml:"tileheight,attr"`
	Width      int                     `xml:"width,attr"`  // Map width in tiles
	Height     int                     `xml:"height,attr"` // Map height in tiles
	Tilesets   []tiledTilesetReference `xml:"tileset"`
	Layers     []tiledLayer            `xml:"layer"` // <layer> elements
}

type tiledTilesetReference struct {
	FirstGid int    `xml:"firstgid,attr"`
	Source   string `xml:"source,attr"`
}

type tiledLayer struct {
	ID     int       `xml:"id,attr"`
	Name   string    `xml:"name,attr"`
	Width  int       `xml:"width,attr"`
	Height int       `xml:"height,attr"`
	Data   tiledData `xml:"data"`
}

type tiledData struct {
	Encoding    string `xml:"encoding,attr"`    // e.g., "csv", "base64"
	Compression string `xml:"compression,attr"` // e.g., "zlib", "gzip" (optional)
	Content     string `xml:",chardata"`        // Captures the inner text content of the <data> tag
}

// tiledTileset maps to the root <tileset> element in a .tsx file
type tiledTileset struct {
	XMLName    xml.Name    `xml:"tileset"`
	TileWidth  int         `xml:"tilewidth,attr"`
	TileHeight int         `xml:"tileheight,attr"`
	Tiles      []tiledTile `xml:"tile"`
	Image      struct {
		Source string `xml:"source,attr"`
		Width  int    `xml:"width,attr"`
		Height int    `xml:"height,attr"`
	} `xml:"image"`
}

// tiledTile maps to a <tile id="..."> element within <tileset>
type tiledTile struct {
	ID          int              `xml:"id,attr"`
	ObjectGroup tiledObjectGroup `xml:"objectgroup"`
}

type tiledObjectGroup struct {
	Objects []tiledObject `xml:"object"`
}

type tiledObject struct {
	ID      int          `xml:"id,attr"`
	Class   string       `xml:"type,attr"`
	Polygon tiledPolygon `xml:"polygon"`
	// X       float64      `xml:"x,attr"`
	// Y       float64      `xml:"y,attr"`
}

type tiledPolygon struct {
	Points string `xml:"points,attr"` // Parsed as space-separated coordinate pairs: "x1,y1 x2,y2 ..."
}

func interpretTMX(r io.Reader) (*tiledMap, error) {
	var tmx tiledMap
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&tmx); err != nil {
		return nil, fmt.Errorf("failed to decode TMX XML: %w", err)
	}
	return &tmx, nil
}

func interpretTSX(name string, r io.Reader) (*tiledTileset, error) {
	var tsx tiledTileset
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&tsx); err != nil {
		return nil, fmt.Errorf("failed to decode TSX XML: %w", err)
	}

	// Check that all polygon collision shapes have counter-clockwise vertex
	// ordering. Correct and warn when they don't, so the issue is caught at
	// load time rather than silently producing wrong collision normals.
	for i := range tsx.Tiles {
		tile := &tsx.Tiles[i]
		for j := range tile.ObjectGroup.Objects {
			obj := &tile.ObjectGroup.Objects[j]
			rawPoints := strings.TrimSpace(obj.Polygon.Points)
			if rawPoints == "" {
				continue
			}

			pairs := strings.Fields(rawPoints)
			type pt struct{ x, y float64 }
			verts := make([]pt, 0, len(pairs))
			for _, pair := range pairs {
				parts := strings.SplitN(pair, ",", 2)
				if len(parts) != 2 {
					continue
				}
				x, err := strconv.ParseFloat(parts[0], 64)
				if err != nil {
					continue
				}
				y, err := strconv.ParseFloat(parts[1], 64)
				if err != nil {
					continue
				}
				verts = append(verts, pt{x, y})
			}

			// Compute signed area using the shoelace formula.
			// Positive area means clockwise winding in screen-space
			// (Y-axis points down), which would produce inverted normals.
			area := 0.0
			n := len(verts)
			for k := 0; k < n; k++ {
				l := (k + 1) % n
				area += verts[k].x*verts[l].y - verts[l].x*verts[k].y
			}
			area /= 2.0

			if area > 0 {
				fmt.Fprintf(os.Stderr,
					"WARNING: tileset %q tile ID %d has clockwise polygon vertices; correcting to counter-clockwise\n",
					name, tile.ID)
				// Reverse the pairs slice to flip winding order.
				for lo, hi := 0, len(pairs)-1; lo < hi; lo, hi = lo+1, hi-1 {
					pairs[lo], pairs[hi] = pairs[hi], pairs[lo]
				}
				obj.Polygon.Points = strings.Join(pairs, " ")
			}
		}
	}

	return &tsx, nil
}
