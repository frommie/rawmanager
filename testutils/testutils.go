// Package testutils provides helper functions for testing image processing functionality.
package testutils

import (
	"fmt"
	"github.com/disintegration/imaging"
	"image/color"
	"os"
	"testing"
)

// CreateEmptyJPEG creates a blank 100x100 white JPEG file at the specified path.
// It is used for testing JPEG processing functions.
func CreateEmptyJPEG(t *testing.T, path string) error {
	t.Helper()
	img := imaging.New(100, 100, color.White)
	return imaging.Save(img, path)
}

// CreateTestXMP creates an XMP sidecar file with the specified rating.
// The XMP file follows Adobe's XMP specification.
func CreateTestXMP(t *testing.T, path string, rating int) error {
	t.Helper()

	xmpContent := fmt.Sprintf(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
            <xmp:Rating>%d</xmp:Rating>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta>`, rating)

	return os.WriteFile(path, []byte(xmpContent), 0644)
}

// CreateTestJPEGWithEmbeddedXMP creates a JPEG file with embedded XMP metadata.
// The rating is stored in the XMP section according to Adobe's specification.
func CreateTestJPEGWithEmbeddedXMP(t *testing.T, path string, rating int) error {
	t.Helper()

	// Create base-JPEG
	img := imaging.New(100, 100, color.White)
	if err := imaging.Save(img, path, imaging.JPEGQuality(90)); err != nil {
		return fmt.Errorf("Error creating JPEG: %v", err)
	}

	// Prepare XMP data
	xmpData := fmt.Sprintf(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
            <xmp:Rating>%d</xmp:Rating>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`, rating)

	// Read existing JPEG
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Error reading JPEG: %v", err)
	}

	// Create APP1 segment with XMP data
	prefix := []byte("http://ns.adobe.com/xap/1.0/\x00")
	xmpLength := len(prefix) + len(xmpData)

	// Rebuild JPEG
	newData := make([]byte, 0, len(data)+xmpLength+4)
	newData = append(newData, 0xFF, 0xD8)                                       // JPEG SOI marker
	newData = append(newData, 0xFF, 0xE1)                                       // APP1 marker
	newData = append(newData, byte((xmpLength+2)>>8), byte((xmpLength+2)&0xFF)) // Length
	newData = append(newData, prefix...)
	newData = append(newData, []byte(xmpData)...)
	newData = append(newData, data[2:]...) // Rest of JPEG data

	// Write new file
	if err := os.WriteFile(path, newData, 0644); err != nil {
		return fmt.Errorf("Error writing JPEG: %v", err)
	}

	return nil
}

// CreateTestDirectory creates a test directory structure with JPEG and RAW files.
// It is used for integration testing of the file processor.
func CreateTestDirectory(t *testing.T, baseDir string, files map[string]int) error {
	t.Helper()

	for filename, rating := range files {
		// Create JPEG with rating
		jpgPath := filename + ".JPG"
		if err := CreateTestJPEGWithEmbeddedXMP(t, jpgPath, rating); err != nil {
			return fmt.Errorf("Error creating %s: %v", jpgPath, err)
		}

		// Create associated RAW file
		rawPath := filename + ".RAF"
		emptyFile, err := os.Create(rawPath)
		if err != nil {
			return fmt.Errorf("Error creating %s: %v", rawPath, err)
		}
		emptyFile.Close()
	}

	return nil
}
