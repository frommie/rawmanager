package xmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/frommie/rawmanager/config"
)

const (
	// namespace identifies APP1 segments containing XMP data
	namespace = "http://ns.adobe.com/xap/1.0/"

	// app1MarkerId is the marker for APP1 segments in JPEG files
	app1MarkerId = 0xE1
)

type XmpMeta struct {
	XMLName xml.Name `xml:"xmpmeta"`
	RDF     struct {
		Description struct {
			Rating       string `xml:"http://ns.adobe.com/xap/1.0/ Rating"`
			RatingAttr   string `xml:"http://ns.adobe.com/xap/1.0/ Rating,attr"`
			MSRating     string `xml:"http://ns.microsoft.com/photo/1.0/ Rating"`
			MSRatingAttr string `xml:"http://ns.microsoft.com/photo/1.0/ Rating,attr"`
		} `xml:"Description"`
	} `xml:"RDF"`
}

// ReadRating reads the rating of a JPEG file from the XMP source selected by mode
func ReadRating(jpgPath string, mode config.XmpMode) (int, error) {
	switch mode {
	case config.XmpModeEmbedded:
		data, err := os.ReadFile(jpgPath)
		if err != nil {
			return 0, err
		}
		xmpData, err := ExtractXmpData(data)
		if err != nil {
			return 0, err
		}
		return GetRating(xmpData)

	case config.XmpModeSeparate:
		// IMG.JPG -> IMG.xmp
		return GetRatingFromSidecar(strings.TrimSuffix(jpgPath, filepath.Ext(jpgPath)) + ".xmp")

	case config.XmpModeSeparateExt:
		// IMG.JPG -> IMG.JPG.xmp
		return GetRatingFromSidecar(jpgPath + ".xmp")

	default:
		return 0, fmt.Errorf("Invalid XMP mode: %s", mode)
	}
}

// IsXmpSegment reports whether a JPEG segment is an APP1 segment containing XMP data
func IsXmpSegment(segment *jpegstructure.Segment) bool {
	return segment.MarkerId == app1MarkerId && bytes.HasPrefix(segment.Data, []byte(namespace))
}

// ExtractXmpData extracts the XMP packet from JPEG data
func ExtractXmpData(jpegData []byte) ([]byte, error) {
	intfc, err := jpegstructure.NewJpegMediaParser().ParseBytes(jpegData)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JPEG file: %v", err)
	}

	for _, segment := range intfc.(*jpegstructure.SegmentList).Segments() {
		if IsXmpSegment(segment) {
			// The payload is the namespace, a null byte and the XMP packet
			return bytes.Trim(segment.Data[len(namespace):], "\x00 \t\r\n"), nil
		}
	}

	return nil, fmt.Errorf("No XMP data found")
}

// GetRating reads the rating from XMP data
func GetRating(xmpData []byte) (int, error) {
	var xmp XmpMeta
	if err := xml.Unmarshal(xmpData, &xmp); err != nil {
		return 0, fmt.Errorf("Error parsing XMP data: %v", err)
	}

	// Ratings can be stored either as element or as attribute
	adobeRating := firstNonEmpty(xmp.RDF.Description.Rating, xmp.RDF.Description.RatingAttr)
	msRatingValue := firstNonEmpty(xmp.RDF.Description.MSRating, xmp.RDF.Description.MSRatingAttr)

	// Check for Adobe XMP Rating first
	if adobeRating != "" {
		rating := 0
		if _, err := fmt.Sscanf(adobeRating, "%d", &rating); err != nil {
			return 0, fmt.Errorf("Error parsing Adobe rating: %v", err)
		}
		return rating, nil
	}

	// If no Adobe rating, check for Microsoft rating
	if msRatingValue != "" {
		var msRating int
		if _, err := fmt.Sscanf(msRatingValue, "%d", &msRating); err != nil {
			return 0, fmt.Errorf("Error parsing Microsoft rating: %v", err)
		}
		return convertMSRating(msRating), nil
	}

	return 0, fmt.Errorf("No rating found in XMP data")
}

// convertMSRating converts a Microsoft rating (0-99) to a standard rating (0-5)
// using the same ranges as Windows: 1-12 = 1, 13-37 = 2, 38-62 = 3, 63-87 = 4, 88-99 = 5
func convertMSRating(msRating int) int {
	if msRating <= 0 {
		return 0
	}
	return min((msRating+12)/25+1, 5)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// GetRatingFromSidecar reads the rating from a separate XMP file
func GetRatingFromSidecar(xmpPath string) (int, error) {
	data, err := os.ReadFile(xmpPath)
	if err != nil {
		return 0, fmt.Errorf("Error reading XMP file: %v", err)
	}

	return GetRating(data)
}
