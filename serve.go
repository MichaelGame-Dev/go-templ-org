package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func serve() {
	// Create a custom handler that serves index.html for directories
	fs := http.FileServer(http.Dir("./public"))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the absolute path to prevent directory traversal
		path := filepath.Join("./public", r.URL.Path)

		// Check if path exists
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
		}

		// If it's a directory, try to serve index.html
		if info != nil && info.IsDir() {
			indexPath := filepath.Join(path, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				// Serve the index.html file
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		// Otherwise, let the file server handle it
		fs.ServeHTTP(w, r)
	})

	fmt.Println("Starting server at http://localhost:8080")
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
