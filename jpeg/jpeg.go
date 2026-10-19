package jpeg

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/xmp"
	"math"
	"os"
	"path/filepath"
)

const (
	// app0MarkerId is the marker for APP0 (JFIF) segments in JPEG files
	app0MarkerId = 0xE0

	// app1MarkerId is the marker for APP1 segments in JPEG files (EXIF and XMP)
	app1MarkerId = 0xE1
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
		return ResizeResult{}, fmt.Errorf("error reading file: %v", err)
	}

	segments, err := parseSegments(data)
	if err != nil {
		return ResizeResult{}, fmt.Errorf("error parsing JPEG file: %v", err)
	}
	exifSegment, xmpSegment := extractMetadata(segments)

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return ResizeResult{}, fmt.Errorf("error decoding image: %v", err)
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
		return ResizeResult{}, fmt.Errorf("error encoding resized image: %v", err)
	}

	output, err := insertMetadata(encoded.Bytes(), exifSegment, xmpSegment)
	if err != nil {
		return ResizeResult{}, err
	}

	if err := writeFileAtomic(jpgPath, output); err != nil {
		return ResizeResult{}, fmt.Errorf("error saving resized JPEG: %v", err)
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

// extractMetadata returns the EXIF and XMP segments of a JPEG
func extractMetadata(segments []*jpegstructure.Segment) (exifSegment, xmpSegment *jpegstructure.Segment) {
	for _, segment := range segments {
		if segment.MarkerId == app1MarkerId {
			if xmp.IsXmpSegment(segment) {
				xmpSegment = segment
			} else {
				exifSegment = segment
			}
		}
	}
	return exifSegment, xmpSegment
}

// insertMetadata adds the EXIF and XMP segments to the encoded JPEG data
func insertMetadata(jpegData []byte, exifSegment, xmpSegment *jpegstructure.Segment) ([]byte, error) {
	segments, err := parseSegments(jpegData)
	if err != nil {
		return nil, fmt.Errorf("error parsing resized image: %v", err)
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
	if exifSegment != nil {
		newSegments = append(newSegments, exifSegment)
	}
	if xmpSegment != nil {
		newSegments = append(newSegments, xmpSegment)
	}
	newSegments = append(newSegments, segments[insertPos:]...)

	var buffer bytes.Buffer
	if err := jpegstructure.NewSegmentList(newSegments).Write(&buffer); err != nil {
		return nil, fmt.Errorf("error serializing JPEG data: %v", err)
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
