package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"log"
	"errors"
	"path/filepath"
	"strings"
)

type XmpMeta struct {
    XMLName xml.Name `xml:"xmpmeta"`
    RDF     struct {
        Description struct {
            Rating string `xml:"http://ns.adobe.com/xap/1.0/ Rating"`
        } `xml:"Description"`
    } `xml:"RDF"`
}

func getRating(xmpPath string) (int, error) {
	data, err := os.ReadFile(xmpPath)
	if err != nil {
		return 0, err
	}

	var xmp XmpMeta
	if err := xml.Unmarshal(data, &xmp); err != nil {
		return 0, err
	}

	rating := 0
	if xmp.RDF.Description.Rating != "" {
		fmt.Sscanf(xmp.RDF.Description.Rating, "%d", &rating)
	}
	return rating, nil
}

func shouldDeleteRAW(jpgPath string) (bool, error) {
    // Prüfe ob JPG existiert
    if _, err := os.Stat(jpgPath); err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return true, nil
        }
        return false, err
    }

    // Korrektur: XMP-Pfad vom Basis-Dateinamen ableiten
    baseName := filepath.Base(jpgPath[:len(jpgPath)-3])
    xmpPath := filepath.Join(filepath.Dir(jpgPath), baseName + "xmp")
		
    if rating, err := getRating(xmpPath); err == nil && (rating == 1 || rating == 2) {
        return true, nil
    } else if err != nil && !errors.Is(err, os.ErrNotExist) {
        return false, err
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
            xmpPath := filepath.Join(parentDir, rawName[:len(rawName)-3] + "xmp")
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

            // Lösche XMP-Datei, falls vorhanden
            if _, err := os.Stat(xmpPath); err == nil {
                fmt.Printf("Lösche XMP-Datei %s\n", xmpPath)
                if err := os.Remove(xmpPath); err != nil {
                    return fmt.Errorf("Fehler beim Löschen von XMP %s: %v", xmpPath, err)
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
	fmt.Println("Starte das Skript...")
	photosDir := os.ExpandEnv("${HOME}/Desktop/Kamera_test/")
	
	err := walkPhotosDir(photosDir)
	if err != nil {
		log.Fatal(err)
	}
}