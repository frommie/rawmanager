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
			Rating   string `xml:"http://ns.adobe.com/xap/1.0/ Rating"`
			MSRating string `xml:"http://ns.microsoft.com/photo/1.0/ Rating"`
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

	// Check for Adobe XMP Rating first
	if xmp.RDF.Description.Rating != "" {
		rating := 0
		if _, err := fmt.Sscanf(xmp.RDF.Description.Rating, "%d", &rating); err != nil {
			return 0, fmt.Errorf("Error parsing Adobe rating: %v", err)
		}
		return rating, nil
	}

	// If no Adobe rating, check for Microsoft rating
	if xmp.RDF.Description.MSRating != "" {
		var msRating int
		if _, err := fmt.Sscanf(xmp.RDF.Description.MSRating, "%d", &msRating); err != nil {
			return 0, fmt.Errorf("Error parsing Microsoft rating: %v", err)
		}
		// Konvertiere Microsoft Rating (0-99) zu Standard Rating (1-5)
		return (msRating + 24) / 25, nil
	}

	return 0, fmt.Errorf("No rating found in XMP data")
}

func GetRatingFromFile(xmpPath string) (int, error) {
	data, err := os.ReadFile(xmpPath)
	if err != nil {
		return 0, fmt.Errorf("Error reading XMP file: %v", err)
	}

	return GetRating(data)
}
