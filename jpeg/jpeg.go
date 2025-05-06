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
	"strings"
)

const (
	// XmpNamespace defines the namespace for Adobe XMP metadata
	xmpNamespace = "http://ns.adobe.com/xap/1.0/"

	// app1MarkerId is the marker for APP1 segments in JPEG files (0xE1)
	app1MarkerId = 0xE1
)

// GetRatingFromFile reads the rating from a JPEG file
func GetRatingFromFile(jpgPath string, cfg *config.Config) (int, error) {
	switch cfg.Xmp.Mode {
	case config.XmpModeEmbedded:
		// Read embedded XMP data from JPEG
		file, err := os.Open(jpgPath)
		if err != nil {
			return 0, err
		}
		defer file.Close()

		xmpData, err := xmp.ExtractXmpData(file)
		if err != nil {
			return 0, err
		}

		return xmp.GetRating(xmpData)

	case config.XmpModeSeparate:
		// Read separate .xmp file
		xmpPath := jpgPath[:len(jpgPath)-len(filepath.Ext(jpgPath))] + ".xmp"
		return xmp.GetRatingFromFile(xmpPath)

	case config.XmpModeSeparateExt:
		// Read .jpg.xmp file
		xmpPath := jpgPath + ".xmp"
		return xmp.GetRatingFromFile(xmpPath)

	default:
		return 0, fmt.Errorf("Invalid XMP mode: %s", cfg.Xmp.Mode)
	}
}

// ResizeWithXMP resizes a JPEG image while preserving XMP and EXIF metadata
func ResizeWithXMP(jpgPath string, config *config.Config, verbose bool) error {
	// Extract metadata from original image
	exifSegment, xmpSegment, err := extractMetadata(jpgPath)
	if err != nil {
		return err
	}

	// Process and resize the image
	newWidth, newHeight, err := resizeImage(jpgPath, config, verbose)
	if err != nil {
		return err
	}

	// If no resize was needed, return early
	if newWidth == 0 && newHeight == 0 {
		return nil
	}

	// Combine resized image with original metadata
	if err := combineImageAndMetadata(jpgPath, exifSegment, xmpSegment, config); err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Image %s resized to %dx%d pixels (EXIF and XMP data preserved)\n",
			jpgPath, newWidth, newHeight)
	}
	return nil
}

// extractMetadata reads EXIF and XMP segments from the original JPEG
func extractMetadata(jpgPath string) (*jpegstructure.Segment, *jpegstructure.Segment, error) {
	data, err := os.ReadFile(jpgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading file: %v", err)
	}

	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(data)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing JPEG file: %v", err)
	}

	sl := intfc.(*jpegstructure.SegmentList)
	var exifSegment, xmpSegment *jpegstructure.Segment

	for _, segment := range sl.Segments() {
		if segment.MarkerId == app1MarkerId {
			if bytes.HasPrefix(segment.Data, []byte(xmpNamespace)) {
				xmpSegment = segment
			} else {
				exifSegment = segment
			}
		}
	}

	return exifSegment, xmpSegment, nil
}

// resizeImage performs the actual image resizing if needed
func resizeImage(jpgPath string, config *config.Config, verbose bool) (int, int, error) {
	img, err := imaging.Open(jpgPath)
	if err != nil {
		return 0, 0, fmt.Errorf("error opening image: %v", err)
	}

	bounds := img.Bounds()
	currentWidth := bounds.Max.X
	currentHeight := bounds.Max.Y
	currentMP := float64(currentWidth*currentHeight) / 1000000.0

	if currentMP <= config.Process.TargetMegapixels {
		if verbose {
			fmt.Printf("Image %s is already small enough (%.1f MP)\n",
				jpgPath, currentMP)
		}
		return 0, 0, nil
	}

	ratio := math.Sqrt(config.Process.TargetMegapixels / currentMP)
	newWidth := int(float64(currentWidth) * ratio)
	newHeight := int(float64(currentHeight) * ratio)

	resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	tempPath := strings.TrimSuffix(jpgPath, config.Files.JpegExtension) + "_temp.jpg"
	if err := imaging.Save(resized, tempPath, imaging.JPEGQuality(config.Process.JpegQuality)); err != nil {
		return 0, 0, fmt.Errorf("error saving temporary image: %v", err)
	}
	defer os.Remove(tempPath)

	return newWidth, newHeight, nil
}

// combineImageAndMetadata combines the resized image with the original metadata
func combineImageAndMetadata(jpgPath string, exifSegment, xmpSegment *jpegstructure.Segment, config *config.Config) error {
	tempPath := strings.TrimSuffix(jpgPath, config.Files.JpegExtension) + "_temp.jpg"

	// Read temporary file
	newData, err := os.ReadFile(tempPath)
	if err != nil {
		return fmt.Errorf("error reading temporary image: %v", err)
	}

	jmp := jpegstructure.NewJpegMediaParser()
	newIntfc, err := jmp.ParseBytes(newData)
	if err != nil {
		return fmt.Errorf("error parsing temporary image: %v", err)
	}

	newSl := newIntfc.(*jpegstructure.SegmentList)
	segments := newSl.Segments()

	// Find insertion position after APP0
	insertPos := 1
	for i, seg := range segments {
		if seg.MarkerId == 0xE0 {
			insertPos = i + 1
			break
		}
	}

	// Create new segment list with metadata
	var newSegments []*jpegstructure.Segment
	newSegments = append(newSegments, segments[:insertPos]...)
	if exifSegment != nil {
		newSegments = append(newSegments, exifSegment)
	}
	if xmpSegment != nil {
		newSegments = append(newSegments, xmpSegment)
	}
	newSegments = append(newSegments, segments[insertPos:]...)

	// Write final file
	newJpeg := jpegstructure.NewSegmentList(newSegments)
	var buffer bytes.Buffer
	if err := newJpeg.Write(&buffer); err != nil {
		return fmt.Errorf("error serializing JPEG data: %v", err)
	}

	if err := os.WriteFile(jpgPath, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("error saving final JPEG: %v", err)
	}

	return nil
}
