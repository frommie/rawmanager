package xmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/dsoprea/go-jpeg-image-structure/v2"
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

// ExtractXmpData extracts XMP data from a file
func ExtractXmpData(file *os.File) ([]byte, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %v", err)
	}

	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JPEG file: %v", err)
	}

	// Get list of segments
	sl := intfc.(*jpegstructure.SegmentList)
	var xmpData []byte

	// Search for the APP1 segment with XMP data
	for _, segment := range sl.Segments() {
		if segment.MarkerId == 0xE1 { // APP1 marker ID is 0xE1
			payload := segment.Data
			if bytes.HasPrefix(payload, []byte("http://ns.adobe.com/xap/1.0/")) {
				// Remove all null bytes from the data
				cleanData := bytes.Map(func(r rune) rune {
					if r == 0 {
						return -1
					}
					return r
				}, payload[len("http://ns.adobe.com/xap/1.0/"):])

				// Remove whitespace at the beginning and end
				xmpData = bytes.TrimSpace(cleanData)
				break
			}
		}
	}

	if xmpData == nil {
		return nil, fmt.Errorf("No XMP data found")
	}

	return xmpData, nil
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

func GetRatingFromFile(xmpPath string) (int, error) {
	data, err := os.ReadFile(xmpPath)
	if err != nil {
		return 0, fmt.Errorf("Error reading XMP file: %v", err)
	}

	return GetRating(data)
}
