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

// Post represents a blog post - add optional metadata fields
type Post struct {
	Title       string
	Date        string
	Description string   // Optional: from #+DESCRIPTION
	Draft       bool     // Optional: from #+DRAFT
	Tags        []string // Optional: from #+FILETAGS
	Slug        string
	URL         string
	Content     string
	FilePath    string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}

	if err := os.RemoveAll("public"); err != nil {
		panic(err)
	}
	if err := os.MkdirAll("public", 0755); err != nil {
		panic(err)
	}

	allPosts, err := loadPosts("posts")
	if err != nil {
		panic(err)
	}

	// Filter out drafts (if #+DRAFT: true)
	var posts []Post
	for _, post := range allPosts {
		if !post.Draft {
			posts = append(posts, post)
		} else {
			fmt.Printf("⏭️  Skipping draft: %s\n", post.Title)
		}
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date > posts[j].Date
	})

	if err := generateIndex(posts); err != nil {
		panic(err)
	}

	for _, post := range posts {
		if err := generatePost(post); err != nil {
			panic(err)
		}
	}

	fmt.Println("✓ Blog generated successfully!")
}

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

func parsePost(filePath string) (Post, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return Post{}, err
	}

	doc := org.New().Parse(bytes.NewReader(content), filePath)

	// Extract metadata from Org headers (all optional except DATE)
	title := doc.Get("TITLE")
	if title == "" {
		title = "Untitled"
	}

	date := doc.Get("DATE")
	if date == "" {
		return Post{}, fmt.Errorf("missing #+DATE: in %s", filePath)
	}

	description := doc.Get("DESCRIPTION")
	draft := strings.ToLower(doc.Get("DRAFT")) == "true"

	var tags []string
	if tagsStr := doc.Get("FILETAGS"); tagsStr != "" {
		tags = strings.Fields(tagsStr)
	}

	// Get slug from #+EXPORT_FILE_NAME or generate from title
	slug := doc.Get("EXPORT_FILE_NAME")
	if slug == "" {
		slug = strings.ToLower(title)
		slug = strings.ReplaceAll(slug, " ", "-")
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return -1
		}, slug)
	}

	url := fmt.Sprintf("/posts/%s/", slug)

	writer := org.NewHTMLWriter()
	htmlContent := writer.WriteNodesAsString(doc.Nodes...)

	return Post{
		Title:       title,
		Date:        date,
		Description: description,
		Draft:       draft,
		Tags:        tags,
		Slug:        slug,
		URL:         url,
		Content:     htmlContent,
		FilePath:    filePath,
	}, nil
}

func generateIndex(posts []Post) error {
	f, err := os.Create("public/index.html")
	if err != nil {
		return err
	}
	defer f.Close()
	return IndexPage(posts).Render(context.Background(), f)
}
func generatePost(post Post) error {
	dir := filepath.Join("public", "posts", post.Slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(dir, "index.html")

	// Use the SEO version
	return GeneratePostWithSEO(post, outputPath)
}
