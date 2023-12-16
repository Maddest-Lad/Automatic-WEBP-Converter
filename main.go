package main

import (
	"fmt"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
	"golang.org/x/image/webp"
)

func main() {
	// Create New Watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("failed to create new watcher: %s", err)
	}
	defer watcher.Close()

	// Start Listening For Events
	go dedupLoop(watcher)

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

	// Block forever
	fmt.Println("WebP Converter Online; Press CTRL+C to Exit")
	<-make(chan struct{})
}

// Deduplicates Events For The Same File
func dedupLoop(w *fsnotify.Watcher) {
	var (
		// Wait 250ms for new events; each new event resets the timer.
		waitFor = 250 * time.Millisecond

		// Keep track of the timers, as path → timer.
		mu     sync.Mutex
		timers = make(map[string]*time.Timer)

		// Callback we run.
		convertToPngEvent = func(event fsnotify.Event) {

			// Check for WebP Files
			if pathIsWebP(event.Name) {
				// Convert to PNG
				if err := convertToPNG(event.Name); err != nil {
					log.Printf("Conversion error: %s", err)
				} else {
					log.Printf("Successfully converted %s to a PNG\n", event.Name)

					if err := sendNotification("Converted"+event.Name, "WebP Converter"); err != nil {
						log.Printf("Notification error: %s", err)
					}
				}
			}

			// Don't need to remove the timer if you don't have a lot of files.
			mu.Lock()
			delete(timers, event.Name)
			mu.Unlock()
		}
	)

	for {
		select {
		// Read from Errors.
		case _, ok := <-w.Errors:
			if !ok { // Channel was closed
				return
			}
		// Read from Events.
		case event, ok := <-w.Events:
			if !ok { // Channel was closed
				return
			}

			// Only Observe File Creation and Writes
			if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
				continue
			}

			// Get timers for the event
			mu.Lock()
			t, ok := timers[event.Name]
			mu.Unlock()

			// No timer yet, so create one.
			if !ok {
				t = time.AfterFunc(math.MaxInt64, func() { convertToPngEvent(event) })
				t.Stop()

				mu.Lock()
				timers[event.Name] = t
				mu.Unlock()
			}

			// Reset the timer for this path, so it will start from 100ms again.
			t.Reset(waitFor)
		}
	}
}

func pathIsWebP(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".webp"
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

func sendNotification(message, title string) error {
	return beeep.Notify(title, message, "resources/icons/icon.png")
}
