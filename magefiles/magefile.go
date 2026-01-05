//go:build mage

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

// Build builds TypeScript and Hugo site
func Build() error {
	mg.Deps(BuildTS)
	fmt.Println("Building Hugo site...")
	return sh.RunV("hugo", "--gc", "--minify")
}

// BuildTS compiles TypeScript to JavaScript
func BuildTS() error {
	fmt.Println("Compiling TypeScript...")

	// Ensure output directory exists
	outDir := "themes/linkpage/static/js"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return sh.RunV("esbuild",
		"src/main.ts",
		"--bundle",
		"--minify",
		"--sourcemap",
		"--target=es2020",
		"--outfile="+outDir+"/main.js",
	)
}

// Serve starts the Hugo development server (builds TS first)
func Serve() error {
	mg.Deps(BuildTS)
	return sh.RunV("hugo", "server", "-D")
}

// Dev runs TypeScript in watch mode alongside Hugo server
func Dev() error {
	mg.Deps(BuildTS)
	fmt.Println("Starting development server...")
	fmt.Println("Note: Run 'mage buildts' after TypeScript changes, or use 'mage watch' in another terminal")
	return sh.RunV("hugo", "server", "-D")
}

// Watch watches TypeScript files and rebuilds on change
func Watch() error {
	fmt.Println("Watching TypeScript files...")
	return sh.RunV("esbuild",
		"src/main.ts",
		"--bundle",
		"--sourcemap",
		"--target=es2020",
		"--outfile=themes/linkpage/static/js/main.js",
		"--watch",
	)
}

// CheckLinks checks for dead links in the built site
func CheckLinks() error {
	mg.Deps(Build)

	fmt.Println("\nChecking for dead links...")

	links, err := extractLinks("public")
	if err != nil {
		return fmt.Errorf("failed to extract links: %w", err)
	}

	if len(links) == 0 {
		fmt.Println("No external links found.")
		return nil
	}

	fmt.Printf("Found %d unique external links to check\n\n", len(links))

	deadLinks := checkLinksParallel(links)

	if len(deadLinks) > 0 {
		fmt.Printf("\n❌ Found %d dead or problematic links:\n", len(deadLinks))
		for _, dl := range deadLinks {
			fmt.Printf("  • %s\n    Status: %s\n", dl.URL, dl.Status)
		}
		return fmt.Errorf("found %d dead links", len(deadLinks))
	}

	fmt.Println("\n✓ All links are valid!")
	return nil
}

type deadLink struct {
	URL    string
	Status string
}

func extractLinks(dir string) ([]string, error) {
	linkSet := make(map[string]bool)
	// Match <a href="..."> links, excluding preconnect/dns-prefetch hints
	anchorRegex := regexp.MustCompile(`<a\s[^>]*href=["']?(https?://[^"'\s>]+)["']?`)
	// Also match links in data attributes for tracking purposes
	dataRegex := regexp.MustCompile(`data-track[^>]*href=["']?(https?://[^"'\s>]+)["']?`)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contentStr := string(content)

		// Extract anchor tag links
		matches := anchorRegex.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				url := strings.TrimSuffix(match[1], `"`)
				url = strings.TrimSuffix(url, `'`)
				linkSet[url] = true
			}
		}

		// Extract data attribute links
		matches = dataRegex.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				url := strings.TrimSuffix(match[1], `"`)
				url = strings.TrimSuffix(url, `'`)
				linkSet[url] = true
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	links := make([]string, 0, len(linkSet))
	for link := range linkSet {
		links = append(links, link)
	}

	return links, nil
}

// skipDomains contains domains with aggressive bot protection that return
// errors to automated checkers but work fine in browsers
var skipDomains = []string{
	"facebook.com",
	"www.facebook.com",
}

func shouldSkipDomain(url string) bool {
	for _, domain := range skipDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

func checkLinksParallel(links []string) []deadLink {
	var (
		deadLinks []deadLink
		mu        sync.Mutex
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, 5) // Limit concurrent requests
	)

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	for _, link := range links {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Skip domains with aggressive bot protection
			if shouldSkipDomain(url) {
				fmt.Printf("  ⊘ %s (skipped - bot protection)\n", url)
				return
			}

			status := checkLink(client, url)
			if status != "" {
				mu.Lock()
				deadLinks = append(deadLinks, deadLink{URL: url, Status: status})
				mu.Unlock()
				fmt.Printf("  ❌ %s\n", url)
			} else {
				fmt.Printf("  ✓ %s\n", url)
			}
		}(link)
	}

	wg.Wait()
	return deadLinks
}

func checkLink(client *http.Client, url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Sprintf("invalid URL: %v", err)
	}

	// Use a realistic browser User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("request failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Consider 2xx and 3xx as valid
	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return ""
}

// Clean removes the public directory
func Clean() error {
	fmt.Println("Cleaning public directory...")
	return os.RemoveAll("public")
}
