package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsoprea/go-jpeg-image-structure/v2"
)

type XmpMeta struct {
	XMLName xml.Name `xml:"xmpmeta"`
	RDF     struct {
		Description struct {
			Rating  string `xml:"http://ns.adobe.com/xap/1.0/ Rating"`
			MSRating string `xml:"http://ns.microsoft.com/photo/1.0/ Rating"`
		} `xml:"Description"`
	} `xml:"RDF"`
}

func getRatingFromJPEG(jpgPath string) (int, error) {
    // Öffne die JPEG-Datei
    file, err := os.Open(jpgPath)
    if err != nil {
        return 0, fmt.Errorf("Fehler beim Öffnen der Datei: %v", err)
    }
    defer file.Close()

    // Lese die JPEG-Datei und analysiere die Segmente
    data, err := os.ReadFile(jpgPath)
    if err != nil {
        return 0, fmt.Errorf("Fehler beim Lesen der Datei: %v", err)
    }

    // Erstelle einen JPEG-Media-Parser
    jmp := jpegstructure.NewJpegMediaParser()
    intfc, err := jmp.ParseBytes(data)
    if err != nil {
        return 0, fmt.Errorf("Fehler beim Parsen der JPEG-Datei: %v", err)
    }

    // Hole die Liste der Segmente und suche nach XMP-Daten
    sl := intfc.(*jpegstructure.SegmentList)
    var xmpData []byte
    for _, segment := range sl.Segments() {
        if segment.MarkerId == 0xE1 { // APP1 marker ID ist 0xE1
            payload := segment.Data
            if bytes.HasPrefix(payload, []byte("http://ns.adobe.com/xap/1.0/")) {
                // Entferne alle Null-Bytes aus den Daten
                cleanData := bytes.Map(func(r rune) rune {
                    if r == 0 {
                        return -1
                    }
                    return r
                }, payload[len("http://ns.adobe.com/xap/1.0/"):])
                
                // Entferne Whitespace am Anfang und Ende
                xmpData = bytes.TrimSpace(cleanData)
                break
            }
        }
    }

    if xmpData == nil {
        return 0, fmt.Errorf("Keine XMP-Daten gefunden")
    }

    // Parse die XMP-Daten
    var xmp XmpMeta
    if err := xml.Unmarshal(xmpData, &xmp); err != nil {
        return 0, fmt.Errorf("Fehler beim Parsen der XMP-Daten: %v", err)
    }

    // Extrahiere die Bewertung
    rating := 0
    if xmp.RDF.Description.Rating != "" {
        // Adobe XMP Rating (1-5)
        fmt.Sscanf(xmp.RDF.Description.Rating, "%d", &rating)
    } else if xmp.RDF.Description.MSRating != "" {
        // Microsoft Photo Rating (0-99)
        var msRating int
        fmt.Sscanf(xmp.RDF.Description.MSRating, "%d", &msRating)
        // Konvertiere Microsoft Rating (0-99) zu Adobe Rating (1-5)
        rating = (msRating + 24) / 25
    }
    
    return rating, nil
}

// Hilfsfunktion für min
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func shouldDeleteRAW(jpgPath string) (bool, error) {
	// Prüfe ob JPG existiert
	if _, err := os.Stat(jpgPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	// Lese Bewertung aus der JPEG-Datei
	rating, err := getRatingFromJPEG(jpgPath)
	if err != nil {
		return false, err
	}

	if rating == 1 || rating == 2 {
		return true, nil
	}

	return false, nil
}

func processRawDirectory(rawDir string, parentDir string) error {
    files, err := os.ReadDir(rawDir)
    if err != nil {
        return err
    }

    for _, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".RAF") {
            rawName := file.Name()
            jpgName := rawName[:len(rawName)-3] + "JPG"
            jpgPath := filepath.Join(parentDir, jpgName)
            shouldDelete, err := shouldDeleteRAW(jpgPath)
            if err != nil {
                return fmt.Errorf("Fehler beim Prüfen von %s: %v", jpgPath, err)
            }

            if shouldDelete {
                rawPath := filepath.Join(rawDir, rawName)
                reason := "keine zugehörige JPG-Datei gefunden"
                if _, err := os.Stat(jpgPath); err == nil {
                    reason = "Bewertung ist 1 oder 2"
                }
                fmt.Printf("Lösche %s (%s)\n", rawPath, reason)
                if err := os.Remove(rawPath); err != nil {
                    return fmt.Errorf("Fehler beim Löschen von %s: %v", rawPath, err)
                }
            }
        }
    }
    return nil
}

func walkPhotosDir(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "raw" {
			parentDir := filepath.Dir(path)
			if err := processRawDirectory(path, parentDir); err != nil {
				return fmt.Errorf("Fehler bei der Verarbeitung von %s: %v", path, err)
			}
		}
		return nil
	})
}

func main() {
    photosDir := os.ExpandEnv("${HOME}/Desktop/Kamera")
	err := walkPhotosDir(photosDir)
	if err != nil {
		log.Fatal(err)
	}
}