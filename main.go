package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/niklasfasching/go-org/org"
)

// Post represents a blog post
type Post struct {
	Title    string
	Date     string
	Slug     string
	URL      string
	Content  string
	FilePath string
}

func main() {
	// Create public directory
	if err := os.RemoveAll("public"); err != nil {
		panic(err)
	}
	if err := os.MkdirAll("public", 0755); err != nil {
		panic(err)
	}
	// Read and parse all posts
	posts, err := loadPosts("posts")
	if err != nil {
		panic(err)
	}

	// Sort posts by date (newest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date > posts[j].Date
	})

	// Generate index page
	if err := generateIndex(posts); err != nil {
		panic(err)
	}

	// Generate individual post pages
	for _, post := range posts {
		if err := generatePost(post); err != nil {
			panic(err)
		}
	}

	fmt.Println("✓ Blog generated successfully!")

	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}
}

// loadPosts reads all .org files from the posts directory
func loadPosts(dir string) ([]Post, error) {
	var posts []Post

	files, err := filepath.Glob(filepath.Join(dir, "*.org"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		post, err := parsePost(file)
		if err != nil {
			fmt.Printf("Warning: skipping %s: %v\n", file, err)
			continue
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// parsePost reads and parses an org file
// parsePost reads and parses an org file
func parsePost(filePath string) (Post, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return Post{}, err
	}

	// Parse org content
	doc := org.New().Parse(bytes.NewReader(content), filePath)

	// Convert to HTML - WriteNodesAsString returns a string directly
	writer := org.NewHTMLWriter()
	htmlContent := writer.WriteNodesAsString(doc.Nodes...)

	// Extract metadata from filename: YYYY-MM-DD-slug.org
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, ".org")
	parts := strings.SplitN(filename, "-", 4)

	if len(parts) < 4 {
		return Post{}, fmt.Errorf("invalid filename format: %s (expected YYYY-MM-DD-slug.org)", filename)
	}

	year, month, day, slug := parts[0], parts[1], parts[2], parts[3]
	date := fmt.Sprintf("%s-%s-%s", year, month, day)
	url := fmt.Sprintf("/%s/%s/%s/%s/", year, month, day, slug)

	// Extract title from org document (first heading)
	title := strings.ReplaceAll(slug, "-", " ") // fallback: use slug as title
	if doc.Nodes != nil && len(doc.Nodes) > 0 {
		for _, node := range doc.Nodes {
			if headline, ok := node.(org.Headline); ok {
				// Extract text from title nodes
				var titleBuf bytes.Buffer
				for _, titleNode := range headline.Title {
					if text, ok := titleNode.(org.Text); ok {
						titleBuf.WriteString(text.Content)
					}
				}
				if titleBuf.Len() > 0 {
					title = strings.TrimSpace(titleBuf.String())
					break
				}
			}
		}
	}

	return Post{
		Title:    title,
		Date:     date,
		Slug:     slug,
		URL:      url,
		Content:  htmlContent,
		FilePath: filePath,
	}, nil
}

// generateIndex creates the index.html file
func generateIndex(posts []Post) error {
	f, err := os.Create("public/index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	// Use context.Background() for templ rendering
	return IndexPage(posts).Render(context.Background(), f)
}

// generatePost creates the HTML file for a single post
func generatePost(post Post) error {
	// Create directory structure: public/YYYY/MM/DD/slug/
	parts := strings.Split(strings.Trim(post.URL, "/"), "/")
	dir := filepath.Join("public", filepath.Join(parts...))

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create index.html in the post directory
	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()

	// Use context.Background() for templ rendering
	return PostPage(post.Title, post.Content).Render(context.Background(), f)
}
