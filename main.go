package main

import (
    "log"
    "os"
    "github.com/frommie/rawmanager/processor"
)

func main() {
    var photosDir string
    
    // Prüfe ob ein Pfad als Parameter übergeben wurde
    if len(os.Args) > 1 {
        photosDir = os.Args[1]
    } else {
        // Verwende aktuelles Arbeitsverzeichnis wenn kein Parameter angegeben
        var err error
        photosDir, err = os.Getwd()
        if err != nil {
            log.Fatal("Fehler beim Ermitteln des aktuellen Verzeichnisses:", err)
        }
    }
    
    proc := &processor.ImageProcessor{
        RootDir: photosDir,
    }
    
    if err := proc.Walk(); err != nil {
        log.Fatal(err)
    }
}