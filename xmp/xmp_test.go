package xmp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetRatingFromFile(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(string) (string, error)
		want      int
		wantErr   bool
	}{
		{
			name: "Rating 4 from XMP",
			setupFunc: func(dir string) (string, error) {
				xmpPath := filepath.Join(dir, "test.xmp")
				err := CreateTestXMP(xmpPath, 4)
				return xmpPath, err
			},
			want:    4,
			wantErr: false,
		},
		{
			name: "Invalid XMP file",
			setupFunc: func(dir string) (string, error) {
				xmpPath := filepath.Join(dir, "invalid.xmp")
				err := os.WriteFile(xmpPath, []byte("invalid"), 0644)
				return xmpPath, err
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tmpDir := t.TempDir()
			xmpPath, err := tt.setupFunc(tmpDir)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute tests
			got, err := GetRatingFromFile(xmpPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRatingFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("GetRatingFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Help function for the tests
func CreateTestXMP(path string, rating int) error {
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
