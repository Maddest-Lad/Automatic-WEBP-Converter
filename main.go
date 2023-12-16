package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/image/webp"
)

func main() {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Listen For Events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) && pathIsWebP(event.Name) {
					log.Printf("Converting %s to a PNG", event.Name)
					if err := convertToPNG(event.Name); err != nil {
						log.Printf("Conversion error: %s", err)
					} else {
						log.Printf("Successfully converted %s to a PNG\n", event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Print("error:", err)
			}
		}
	}()

	// Add Directories to Watch
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln("Unable to find home directory:", err)
	}

	// Todo replace with Custom Config File and/or UI
	downloads := filepath.Join(homeDirectory, "Downloads")
	pictures := filepath.Join(homeDirectory, "Pictures")

	err = watcher.Add(downloads)
	if err != nil {
		log.Printf("failed to watch %s %s", downloads, err)
	}

	err = watcher.Add(pictures)
	if err != nil {
		log.Printf("failed to watch %s %s", pictures, err)
	}

	// Block main goroutine forever.
	<-make(chan struct{})
}

func pathIsWebP(path string) bool {
	ext := filepath.Ext(path)
	ext = strings.TrimPrefix(ext, ".")
	fmt.Println(path, ext)
	return ext == "webp"
}

func convertToPNG(path string) error {
	directory := strings.TrimSuffix(path, filepath.Base(path))
	filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	outputPath := filepath.Join(directory, filename+".png")

	// Load Image to Convert
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to Open the File: %v", err)
	}
	defer f.Close()

	// Decode the WebP
	webpFile, err := webp.Decode(f)
	if err != nil {
		return fmt.Errorf("webp decode failed: %v", err)
	}

	// Create PNG
	pngFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("png creation failed: %v", err)
	}
	defer pngFile.Close()

	// Encode PNG
	err = png.Encode(pngFile, webpFile)
	if err != nil {
		return fmt.Errorf("png encode failed: %v", err)
	}
	return nil
}
