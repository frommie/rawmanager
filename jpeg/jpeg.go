package jpeg

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/frommie/rawmanager/config"
)

const (
	// Markers of the application segments APP0 (JFIF), APP2 (e.g. ICC profile, MPF) and APP15
	app0MarkerId  = 0xE0
	app2MarkerId  = 0xE2
	app15MarkerId = 0xEF

	// comMarkerId is the marker for comment segments
	comMarkerId = 0xFE

	// mpfIdentifier identifies APP2 segments in the Multi-Picture Format
	mpfIdentifier = "MPF\x00"
)

// ResizeResult describes the outcome of ResizeWithXMP
type ResizeResult struct {
	Resized bool // false if the image was already small enough
	Width   int  // width of the resulting image
	Height  int  // height of the resulting image
}

// ResizeWithXMP resizes a JPEG image to the target size while preserving XMP and EXIF metadata.
// The original file is replaced atomically, so it stays intact if anything fails.
func ResizeWithXMP(jpgPath string, cfg config.ProcessConfig) (ResizeResult, error) {
	data, err := os.ReadFile(jpgPath)
	if err != nil {
		return ResizeResult{}, fmt.Errorf("error reading file: %w", err)
	}

	segments, err := parseSegments(data)
	if err != nil {
		return ResizeResult{}, fmt.Errorf("error parsing JPEG file: %w", err)
	}
	metadata := metadataSegments(segments)

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return ResizeResult{}, fmt.Errorf("error decoding image: %w", err)
	}

	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	currentMP := float64(width*height) / 1000000.0
	if currentMP <= cfg.TargetMegapixels {
		return ResizeResult{Width: width, Height: height}, nil
	}

	ratio := math.Sqrt(cfg.TargetMegapixels / currentMP)
	newWidth := int(float64(width) * ratio)
	newHeight := int(float64(height) * ratio)
	resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	var encoded bytes.Buffer
	if err := imaging.Encode(&encoded, resized, imaging.JPEG, imaging.JPEGQuality(cfg.JpegQuality)); err != nil {
		return ResizeResult{}, fmt.Errorf("error encoding resized image: %w", err)
	}

	output, err := insertMetadata(encoded.Bytes(), metadata)
	if err != nil {
		return ResizeResult{}, err
	}

	if err := writeFileAtomic(jpgPath, output); err != nil {
		return ResizeResult{}, fmt.Errorf("error saving resized JPEG: %w", err)
	}

	return ResizeResult{Resized: true, Width: newWidth, Height: newHeight}, nil
}

func parseSegments(data []byte) ([]*jpegstructure.Segment, error) {
	intfc, err := jpegstructure.NewJpegMediaParser().ParseBytes(data)
	if err != nil {
		return nil, err
	}
	return intfc.(*jpegstructure.SegmentList).Segments(), nil
}

// metadataSegments returns all segments of a JPEG that have to be preserved when re-encoding it:
// APP1-APP15 (e.g. EXIF, XMP, ICC profile) and comments. APP0 (JFIF) is written by the encoder and
// MPF is dropped, as its offsets to the embedded preview images would be invalid after re-encoding.
func metadataSegments(segments []*jpegstructure.Segment) []*jpegstructure.Segment {
	var result []*jpegstructure.Segment
	for _, segment := range segments {
		isApp := segment.MarkerId > app0MarkerId && segment.MarkerId <= app15MarkerId
		isMpf := segment.MarkerId == app2MarkerId && bytes.HasPrefix(segment.Data, []byte(mpfIdentifier))
		if (isApp && !isMpf) || segment.MarkerId == comMarkerId {
			result = append(result, segment)
		}
	}
	return result
}

// insertMetadata adds the metadata segments to the encoded JPEG data
func insertMetadata(jpegData []byte, metadata []*jpegstructure.Segment) ([]byte, error) {
	segments, err := parseSegments(jpegData)
	if err != nil {
		return nil, fmt.Errorf("error parsing resized image: %w", err)
	}

	// Insert after APP0 if present, otherwise directly after SOI
	insertPos := 1
	for i, seg := range segments {
		if seg.MarkerId == app0MarkerId {
			insertPos = i + 1
			break
		}
	}

	var newSegments []*jpegstructure.Segment
	newSegments = append(newSegments, segments[:insertPos]...)
	newSegments = append(newSegments, metadata...)
	newSegments = append(newSegments, segments[insertPos:]...)

	var buffer bytes.Buffer
	if err := jpegstructure.NewSegmentList(newSegments).Write(&buffer); err != nil {
		return nil, fmt.Errorf("error serializing JPEG data: %w", err)
	}
	return buffer.Bytes(), nil
}

// writeFileAtomic replaces the file at path with data, keeping its permissions.
// The data is written to a temporary file in the same directory which is then renamed.
func writeFileAtomic(path string, data []byte) (err error) {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmp.Name(), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
