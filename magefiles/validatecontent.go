//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/magefile/mage/mg"
	"gopkg.in/yaml.v3"
)

var validProjectStatuses = map[string]bool{
	"under-construction": true,
	"funded":             true,
	"in-design":          true,
	"proposed":           true,
	"completed":          true,
}

// ValidateContent checks front matter on all project pages for required fields and valid status values.
func ValidateContent() error {
	fmt.Println("Validating project front matter...")
	var failures []string

	err := filepath.Walk("content/projects", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		if filepath.Base(path) == "_index.md" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		fm, err := parseFrontMatter(raw)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		for _, field := range []string{"title", "description", "status", "agency"} {
			v, ok := fm[field]
			if !ok || strings.TrimSpace(fmt.Sprintf("%v", v)) == "" {
				failures = append(failures, fmt.Sprintf("%s: missing required field %q", path, field))
			}
		}

		if raw, ok := fm["status"]; ok {
			if s := strings.TrimSpace(fmt.Sprintf("%v", raw)); s != "" && !validProjectStatuses[s] {
				failures = append(failures, fmt.Sprintf(
					"%s: invalid status %q — must be one of: under-construction, funded, in-design, proposed, completed", path, s))
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Printf("  ❌ %s\n", f)
		}
		return fmt.Errorf("found %d front matter issue(s)", len(failures))
	}

	fmt.Println("✓ All project front matter is valid")
	return nil
}

func parseFrontMatter(content []byte) (map[string]interface{}, error) {
	s := string(content)
	if !strings.HasPrefix(s, "---") {
		return nil, fmt.Errorf("no front matter delimiter")
	}
	parts := strings.SplitN(s, "\n", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("empty front matter")
	}
	end := strings.Index(parts[1], "\n---")
	if end < 0 {
		return nil, fmt.Errorf("front matter closing delimiter not found")
	}
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(parts[1][:end]), &fm); err != nil {
		return nil, err
	}
	return fm, nil
}

// CheckInternalLinks validates that all internal links in the built site resolve to real pages.
func CheckInternalLinks() error {
	mg.Deps(Build)

	fmt.Println("\nChecking internal links...")

	var broken []struct{ file, link string }
	seen := make(map[string]bool)
	hrefRe := regexp.MustCompile(`href="(/[^"]*)"`)

	err := filepath.Walk("public", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, m := range hrefRe.FindAllStringSubmatch(string(content), -1) {
			link := m[1]
			// strip fragment and query string
			if i := strings.IndexAny(link, "#?"); i >= 0 {
				link = link[:i]
			}
			if link == "" {
				continue
			}
			key := path + "|" + link
			if seen[key] {
				continue
			}
			seen[key] = true

			if !internalPathExists(link) {
				broken = append(broken, struct{ file, link string }{path, link})
				fmt.Printf("  ❌ %s → %s\n", path, link)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(broken) > 0 {
		fmt.Printf("\n❌ Found %d broken internal link(s):\n", len(broken))
		for _, b := range broken {
			fmt.Printf("  • %s\n    Link: %s\n", b.file, b.link)
		}
		return fmt.Errorf("found %d broken internal links", len(broken))
	}

	fmt.Println("\n✓ All internal links are valid!")
	return nil
}

// internalPathExists checks whether a URL path maps to a file in public/.
func internalPathExists(link string) bool {
	// Exact match (static files: /js/main.js, /events.ics, etc.)
	if _, err := os.Stat("public" + link); err == nil {
		return true
	}
	// Directory index (/foo/ → public/foo/index.html, /foo → public/foo/index.html)
	if _, err := os.Stat("public" + strings.TrimSuffix(link, "/") + "/index.html"); err == nil {
		return true
	}
	return false
}
