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
)

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
	// Social platforms
	"facebook.com",
	"www.facebook.com",
	"instagram.com",
	"www.instagram.com",
	"bsky.app",
	// Government sites that block automated checks
	"beavertonoregon.gov",
	"www.beavertonoregon.gov",
	"apps2.beavertonoregon.gov",
	"tigard-or.gov",
	"www.tigard-or.gov",
	"engage.tigard-or.gov",
	"www.engage.tigard-or.gov",
	"hillsboro-oregon.gov",
	"www.hillsboro-oregon.gov",
	"tualatinoregon.gov",
	"www.tualatinoregon.gov",
	"washingtoncountyor.gov",
	"www.washingtoncountyor.gov",
	"oregon.gov",
	"www.oregon.gov",
	"oregonmetro.gov",
	"www.oregonmetro.gov",
	"trimet.org",
	"www.trimet.org",
	"thprd.org",
	"www.thprd.org",
	"tualatinhillsparks.org",
	"www.tualatinhillsparks.org",
	"portland.gov",
	"www.portland.gov",
	"sherwoodoregon.gov",
	"www.sherwoodoregon.gov",
	"ci.cornelius.or.us",
	"www.ci.cornelius.or.us",
	"ci.king-city.or.us",
	"www.ci.king-city.or.us",
	// Advocacy / news sites that block bots
	"bikeloudpdx.org",
	"www.bikeloudpdx.org",
	"thestreettrust.org",
	"www.thestreettrust.org",
	"oregonwalks.org",
	"www.oregonwalks.org",
	"bikeportland.org",
	"www.bikeportland.org",
	"washcobtc.org",
	"www.washcobtc.org",
	"uniteoregon.org",
	"www.uniteoregon.org",
	"oregontrailscoalition.org",
	"www.oregontrailscoalition.org",
	"swtrails.org",
	"www.swtrails.org",
	"wta-tma.org",
	"www.wta-tma.org",
	"ridewithgps.com",
	"www.ridewithgps.com",
	"providence.org",
	"www.providence.org",
	"hillsboronewstimes.com",
	"www.hillsboronewstimes.com",
	"northplains.gov",
	"www.northplains.gov",
	"forestgrove-or.gov",
	"www.forestgrove-or.gov",
	"durham-oregon.us",
	"www.durham-oregon.us",
	"pridebeaverton.org",
	"www.pridebeaverton.org",
	"banksoregon.gov",
	"www.banksoregon.gov",
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

// shift2bikes event URL pattern: https://shift2bikes.org/calendar/event-XXXXX
var shift2bikesEventRegex = regexp.MustCompile(`^https?://(?:www\.)?shift2bikes\.org/calendar/event-(\d+)`)

func checkLink(client *http.Client, url string) string {
	// For shift2bikes event pages, check the API directly since the
	// page is a client-side SPA that won't show errors in raw HTML
	if m := shift2bikesEventRegex.FindStringSubmatch(url); m != nil {
		return checkShift2bikesEvent(client, m[1])
	}

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

func checkShift2bikesEvent(client *http.Client, eventID string) string {
	apiURL := "https://shift2bikes.org/api/events.php?id=" + eventID
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Sprintf("invalid URL: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("request failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// 403 from the Shift2Bikes API typically means rate-limiting during bulk
	// checks, not that the event is gone. Only flag 404 as truly invalid.
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Sprintf("shift2bikes event %s not found (HTTP 404)", eventID)
	}
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusForbidden {
		return fmt.Sprintf("shift2bikes event %s error (API returned HTTP %d)", eventID, resp.StatusCode)
	}

	return ""
}
