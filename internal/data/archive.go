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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/griffithsh/magewars/embedded"
	"github.com/griffithsh/magewars/internal/game"
)

// Archive is a store of game data.
type Archive struct {
	images     map[string]image.Image
	animations map[string][]game.Frame
}

// NewArchive constructs a new Archive.
func NewArchive() (*Archive, error) {
	archive := Archive{
		images:     map[string]image.Image{},
		animations: map[string][]game.Frame{},
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
	default:
		fmt.Fprintf(os.Stderr, "WARNING: no handler configured for %q, detected extension was %q\n", filename, getExt(filename))
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
