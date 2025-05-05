package main

import (
	"flag"
	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/processor"
	"log"
	"os"
)

func main() {
	var (
		photosDir  string
		configPath string
		verbose    bool
	)

	flag.StringVar(&configPath, "config", "", "Path to YAML configuration file")
	flag.BoolVar(&verbose, "v", false, "Verbose mode (shows detailed output)")
	flag.Parse()

	// Check if a path was passed as an argument
	args := flag.Args()
	if len(args) > 0 {
		photosDir = args[0]
	} else {
		var err error
		photosDir, err = os.Getwd()
		if err != nil {
			log.Fatal("Error determining current directory:", err)
		}
	}

	// Load configuration
	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			log.Fatal("Error loading configuration:", err)
		}
	} else {
		cfg = config.NewDefaultConfig()
	}

	proc := &processor.ImageProcessor{
		RootDir: photosDir,
		Config:  cfg,
		Verbose: verbose,
	}

	if err := proc.Process(); err != nil {
		log.Fatal(err)
	}
}
