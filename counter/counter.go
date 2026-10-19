package counter

import (
	"os"
	"path/filepath"

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
			if config.Files.IsJpeg(info.Name()) {
				c.JpegCount++
			}
			// Count RAWs
			if config.Files.IsRaw(info.Name()) {
				c.RawCount++
			}
		}
		return nil
	})
}
