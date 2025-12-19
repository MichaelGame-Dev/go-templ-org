package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
)

// GeneratePostWithSEO generates a post page with SEO metadata
func GeneratePostWithSEO(post Post, outputPath string) error {
	fmt.Printf("\n🔧 GeneratePostWithSEO called for: %s\n", post.Title)

	var buf bytes.Buffer

	if err := PostPage(post).Render(context.Background(), &buf); err != nil {
		return err
	}

	html := buf.String()
	fmt.Printf("   Original HTML length: %d bytes\n", len(html))

	html = InjectSEOMetadata(html, post)
	fmt.Printf("   Modified HTML length: %d bytes\n", len(html))

	return os.WriteFile(outputPath, []byte(html), 0644)
}

// InjectSEOMetadata adds SEO meta tags to HTML
func InjectSEOMetadata(html string, post Post) string {
	fmt.Printf("\n💉 InjectSEOMetadata called\n")
	fmt.Printf("   Title: %s\n", post.Title)
	fmt.Printf("   Description: %s\n", post.Description)
	fmt.Printf("   Tags: %v\n", post.Tags)

	var metaTags strings.Builder
	metaTags.WriteString("\n")

	if post.Description != "" {
		metaTags.WriteString(fmt.Sprintf(`<meta name="description" content="%s">`, post.Description))
		metaTags.WriteString("\n")
	}

	if len(post.Tags) > 0 {
		metaTags.WriteString(fmt.Sprintf(`<meta name="keywords" content="%s">`, strings.Join(post.Tags, ", ")))
		metaTags.WriteString("\n")
	}

	metaTags.WriteString(`<meta property="og:type" content="article">`)
	metaTags.WriteString("\n")
	metaTags.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s">`, post.Title))
	metaTags.WriteString("\n")

	if post.Description != "" {
		metaTags.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s">`, post.Description))
		metaTags.WriteString("\n")
	}

	metaTags.WriteString(`<meta property="twitter:card" content="summary_large_image">`)
	metaTags.WriteString("\n")
	metaTags.WriteString(fmt.Sprintf(`<meta property="twitter:title" content="%s">`, post.Title))
	metaTags.WriteString("\n")

	if post.Description != "" {
		metaTags.WriteString(fmt.Sprintf(`<meta property="twitter:description" content="%s">`, post.Description))
		metaTags.WriteString("\n")
	}

	fmt.Printf("   Meta tags to inject:\n%s\n", metaTags.String())

	// Check if </title> exists
	if !strings.Contains(html, "</title>") {
		fmt.Println("   ✗ ERROR: </title> tag not found in HTML!")
		return html
	}

	fmt.Println("   ✓ Found </title> tag")

	// Insert after </title> tag
	result := strings.Replace(html,
		"</title>",
		"</title>"+metaTags.String(),
		1)

	if result == html {
		fmt.Println("   ✗ ERROR: String replacement didn't work!")
	} else {
		fmt.Println("   ✓ Successfully injected SEO tags")
	}

	return result
}
