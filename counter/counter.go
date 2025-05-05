package counter

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/frommie/rawmanager/config"
)

type FileCounter struct {
	JpegCount int
	RawCount  int
}

func (c *FileCounter) CountFiles(rootDir string, config *config.Config) error {
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Überspringe Fehler
		}

		if !info.IsDir() {
			// Count JPEGs
			if strings.HasSuffix(strings.ToUpper(info.Name()), config.Files.JpegExtension) {
				c.JpegCount++
			}
			// Count RAWs
			if strings.HasSuffix(strings.ToUpper(info.Name()), config.Files.RawExtension) {
				c.RawCount++
			}
		}
		return nil
	})
}
