package main

import (
    "log"
    "os"
    "github.com/frommie/rawmanager/processor"
)

func main() {
    photosDir := os.ExpandEnv("${HOME}/Desktop/Kamera_test")
    
    proc := &processor.ImageProcessor{
        RootDir: photosDir,
    }
    
    if err := proc.Walk(); err != nil {
        log.Fatal(err)
    }
}