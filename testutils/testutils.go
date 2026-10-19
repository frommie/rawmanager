// Package testutils provides helper functions for testing image processing functionality.
package testutils

import (
	"fmt"
	"image/color"
	"os"
	"testing"

	"github.com/disintegration/imaging"
)

// xmpPacket returns an XMP packet with the specified rating following Adobe's XMP specification.
func xmpPacket(rating int) string {
	return fmt.Sprintf(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
            <xmp:Rating>%d</xmp:Rating>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`, rating)
}

// CreateEmptyJPEG creates a blank 100x100 white JPEG file at the specified path.
func CreateEmptyJPEG(t *testing.T, path string) error {
	t.Helper()
	img := imaging.New(100, 100, color.White)
	return imaging.Save(img, path)
}

// CreateTestXMP creates an XMP sidecar file with the specified rating.
func CreateTestXMP(t *testing.T, path string, rating int) error {
	t.Helper()
	return os.WriteFile(path, []byte(xmpPacket(rating)), 0644)
}

// CreateTestJPEGWithEmbeddedXMP creates a blank 100x100 JPEG file with embedded XMP rating.
func CreateTestJPEGWithEmbeddedXMP(t *testing.T, path string, rating int) error {
	t.Helper()
	if err := CreateEmptyJPEG(t, path); err != nil {
		return fmt.Errorf("error creating JPEG: %w", err)
	}
	return EmbedXMP(t, path, rating)
}

// EmbedXMP inserts an APP1 segment with XMP rating data into an existing JPEG file.
func EmbedXMP(t *testing.T, path string, rating int) error {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading JPEG: %w", err)
	}

	// APP1 segment: namespace, null byte and XMP packet
	payload := append([]byte("http://ns.adobe.com/xap/1.0/\x00"), xmpPacket(rating)...)
	length := len(payload) + 2 // length field includes itself

	newData := make([]byte, 0, len(data)+len(payload)+4)
	newData = append(newData, 0xFF, 0xD8)                         // SOI marker
	newData = append(newData, 0xFF, 0xE1)                         // APP1 marker
	newData = append(newData, byte(length>>8), byte(length&0xFF)) // segment length
	newData = append(newData, payload...)
	newData = append(newData, data[2:]...) // rest of the original JPEG after SOI

	if err := os.WriteFile(path, newData, 0644); err != nil {
		return fmt.Errorf("error writing JPEG: %w", err)
	}
	return nil
}
