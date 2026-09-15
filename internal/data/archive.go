package data

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/griffithsh/magewars/embedded"
)

// Archive is a store of game data.
type Archive struct {
	images map[string]image.Image
}

// NewArchive constructs a new Archive.
func NewArchive() (*Archive, error) {
	archive := Archive{
		images: map[string]image.Image{},
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
