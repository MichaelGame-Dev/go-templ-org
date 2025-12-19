package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/fsnotify/fsnotify"
)

func serve() {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Watch the posts directory
	if err := watcher.Add("posts"); err != nil {
		log.Fatal(err)
	}

	// Watch for .templ file changes (for homepage updates)
	if err := watcher.Add("."); err != nil {
		log.Fatal(err)
	}

	// Channel for reload events
	reloadChan := make(chan bool, 10) // Buffered channel

	// Watch for file changes
	// Watch for file changes
	go func() {
		debounce := time.NewTimer(100 * time.Millisecond)
		debounce.Stop()
		var lastEvent fsnotify.Event

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Watch for .org and .templ file changes
				if event.Op&fsnotify.Write == fsnotify.Write {
					ext := filepath.Ext(event.Name)
					if ext == ".org" || ext == ".templ" {
						fmt.Printf("📝 File changed: %s\n", event.Name)
						lastEvent = event
						debounce.Reset(100 * time.Millisecond)
					}
				}
			case <-debounce.C:
				fmt.Println("🔄 Regenerating site...")

				// If a .templ file changed, run templ generate first
				if filepath.Ext(lastEvent.Name) == ".templ" {
					fmt.Println("📦 Running templ generate...")
					cmd := exec.Command("templ", "generate")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						fmt.Printf("❌ Error running templ generate: %v\n", err)
						continue
					}
				}

				if err := regenerateSite(); err != nil {
					fmt.Printf("❌ Error regenerating: %v\n", err)
				} else {
					fmt.Println("✓ Site regenerated")
					// Send reload signal to all connected browsers
					select {
					case reloadChan <- true:
					default:
						// Channel full, skip
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	// SSE endpoint for browser reload
	http.HandleFunc("/_reload/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send initial connection message
		fmt.Fprintf(w, "data: connected\n\n")
		flusher.Flush()

		for {
			select {
			case <-reloadChan:
				fmt.Fprintf(w, "data: reload\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			case <-time.After(30 * time.Second):
				// Send keepalive
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	})

	// File server with index.html handling
	fs := http.FileServer(http.Dir("./public"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join("./public", r.URL.Path)
		info, err := os.Stat(path)

		if err == nil && info.IsDir() {
			indexPath := filepath.Join(path, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		fs.ServeHTTP(w, r)
	})

	fmt.Println("🚀 Starting server at http://localhost:8080")
	fmt.Println("👀 Watching for changes in .org and .templ files...")
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func regenerateSite() error {
	posts, err := loadPosts("posts")
	if err != nil {
		return err
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date > posts[j].Date
	})

	if err := generateIndex(posts); err != nil {
		return err
	}

	for _, post := range posts {
		if err := generatePost(post); err != nil {
			return err
		}
	}

	return nil
}
