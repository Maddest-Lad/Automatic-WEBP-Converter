package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	"golang.org/x/image/webp"
)

func main() {
	// Create Watcher and Filter to File Creation Events
	w := watcher.New()
	w.FilterOps(watcher.Create)

	// Filter to WebP Files
	w.AddFilterHook(IncludeExtensionsFilter([]string{"webp"}))

	go func() {
		for {
			select {
			case event := <-w.Event:
				if err := convertToPNG(event.Path); err != nil {
					log.Println("Conversion error:", err)
				} else {
					log.Printf("Successfully converted %s to a PNG\n", event.Path)
				}

			case err := <-w.Error:
				log.Println("Watcher error:", err)

			case <-w.Closed:
				return
			}
		}
	}()

	// Add Directories to Watch
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln("Unable to find home directory:", err)
	}

	w.Add(filepath.Join(homeDirectory, "Downloads"))
	w.Add(filepath.Join(homeDirectory, "Pictures"))

	fmt.Println()

	// Start the watching process, check for changes every second.
	if err := w.Start(time.Second); err != nil {
		log.Fatalln(err)
	}
}

// Accepts or Rejects a File Based on it's Extension
func IncludeExtensionsFilter(extensions []string) watcher.FilterFileHookFunc {
	return func(info os.FileInfo, fullPath string) error {
		ext := filepath.Ext(info.Name())
		ext = strings.TrimPrefix(ext, ".")

		// Check if the file's extension is in the list of allowed extensions
		for _, allowedExt := range extensions {
			if ext == allowedExt {
				return nil
			}
		}
		// No matching extension.
		return watcher.ErrSkip
	}
}

func convertToPNG(fullpath string) error {
	directory := strings.TrimSuffix(fullpath, filepath.Base(fullpath))
	filename := strings.TrimSuffix(filepath.Base(fullpath), filepath.Ext(fullpath))
	outputPath := filepath.Join(directory, filename+".png")

	// Load Image to Convert
	f, err := os.Open(fullpath)
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
