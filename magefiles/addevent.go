//go:build mage

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// AddEvent creates a new event, optionally posts it to Shift2Bikes, and appends it to events.md.
//
// All prompts can be pre-filled via environment variables for non-interactive use.
// In a non-interactive terminal, any missing required variable causes exit 1.
//
// Environment variables:
//
//	EVENT_TYPE         beaverton, tigard, or custom
//	EVENT_DATE         MM/DD or MM/DD/YYYY
//	EVENT_TITLE        event title (custom only)
//	EVENT_DETAILS      event description (custom only)
//	EVENT_TIME         start time HH:MM:SS (custom only)
//	EVENT_TIME_DETAILS display time e.g. "10am to 2pm" (custom only)
//	EVENT_VENUE        venue name (custom only)
//	EVENT_ADDRESS      address (custom only)
//	EVENT_AREA         area code N/NE/NW/SE/SW/E/W (custom only)
//	EVENT_LOC_DETAILS  location details (custom only)
//	EVENT_START        start location for events.md
//	EVENT_END          end location for events.md
//	EVENT_SHIFT_MODE   create, existing, or skip
//	EVENT_SHIFT_URL    existing Shift2Bikes calendar URL
//	EVENT_ROUTE        RideWithGPS route URL (optional)
//	EVENT_SECTION      YAML comment text to insert after (optional)
//	EVENT_CONFIRM      yes to skip confirmation prompt
func AddEvent() error {
	// Choose event type
	eventTypes := []string{"Beaverton Happy Hour", "Tigard Happy Hour", "Custom Event"}
	typeIdx, err := resolveChoice("EVENT_TYPE", "Select event type", eventTypes)
	if err != nil {
		return err
	}

	// Collect event details based on type
	var entry eventEntry
	var payload *shift2bikesPayload
	switch typeIdx {
	case 0:
		entry, payload, err = collectBeavertonHappyHour()
	case 1:
		entry, payload, err = collectTigardHappyHour()
	case 2:
		entry, payload, err = collectCustomEvent()
	}
	if err != nil {
		return err
	}

	// Handle Shift2Bikes integration
	shiftURL, err := handleShift2Bikes(payload, typeIdx)
	if err != nil {
		return err
	}
	if shiftURL != "" {
		entry.URL = shiftURL
	}

	// Optional route URL
	route, err := resolveOptional("EVENT_ROUTE", "RideWithGPS route URL (optional, press enter to skip)")
	if err != nil {
		return err
	}
	entry.Route = route

	// Summary and confirmation
	printEventSummary(entry)

	confirmed, err := resolveConfirm("EVENT_CONFIRM", "Add this event to events.md?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Canceled.")
		return nil
	}

	// Determine which section to insert into
	section, err := determineSection(typeIdx)
	if err != nil {
		return err
	}

	// Append to events.md
	eventsPath := "content/events.md"
	if err := appendEventToFile(eventsPath, entry, section); err != nil {
		return fmt.Errorf("failed to update %s: %w", eventsPath, err)
	}
	fmt.Printf("\nEvent added to %s\n", eventsPath)

	// Social media templates
	printSocialTemplates(entry)

	return nil
}

// --- Data structures ---

type eventEntry struct {
	Title string
	Date  string // display format: "January 2, 2026"
	URL   string
	Route string
	Start string
	End   string
}

type shift2bikesPayload struct {
	ID            string       `json:"id"`
	Secret        string       `json:"secret"`
	Title         string       `json:"title"`
	Details       string       `json:"details"`
	Audience      string       `json:"audience"`
	Time          string       `json:"time"`
	TimeDetails   string       `json:"timedetails"`
	EventDuration string       `json:"eventduration"`
	Area          string       `json:"area"`
	Venue         string       `json:"venue"`
	Address       string       `json:"address"`
	LocDetails    string       `json:"locdetails"`
	LocEnd        string       `json:"locend"`
	Length        string       `json:"length"`
	Organizer     string       `json:"organizer"`
	Email         string       `json:"email"`
	HideEmail     string       `json:"hideemail"`
	WebName       string       `json:"webname"`
	WebURL        string       `json:"weburl"`
	Phone         string       `json:"phone"`
	Contact       string       `json:"contact"`
	TinyTitle     string       `json:"tinytitle"`
	PrintDescr    string       `json:"printdescr"`
	CodeOfConduct string       `json:"code_of_conduct"`
	ReadComic     string       `json:"read_comic"`
	DateStatuses  []dateStatus `json:"datestatuses"`
}

type dateStatus struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	Status    string `json:"status"`
	Newsflash string `json:"newsflash"`
}

type shift2bikesResponse struct {
	DateStatuses []struct {
		ID string `json:"id"`
	} `json:"datestatuses"`
}

// --- Interactive / non-interactive helpers ---

var stdinScanner *bufio.Scanner

func scanner() *bufio.Scanner {
	if stdinScanner == nil {
		stdinScanner = bufio.NewScanner(os.Stdin)
	}
	return stdinScanner
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func resolveValue(envVar, prompt, defaultVal string) (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}
	if !isInteractive() {
		return "", fmt.Errorf("non-interactive: set %s environment variable", envVar)
	}
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	scanner().Scan()
	val := strings.TrimSpace(scanner().Text())
	if val == "" && defaultVal != "" {
		return defaultVal, nil
	}
	if val == "" {
		return "", fmt.Errorf("%s is required", prompt)
	}
	return val, nil
}

func resolveOptional(envVar, prompt string) (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}
	if !isInteractive() {
		return "", nil
	}
	fmt.Printf("%s: ", prompt)
	scanner().Scan()
	return strings.TrimSpace(scanner().Text()), nil
}

func resolveChoice(envVar, prompt string, options []string) (int, error) {
	if envVar != "" && os.Getenv(envVar) != "" {
		v := os.Getenv(envVar)
		v = strings.ToLower(v)
		for i, opt := range options {
			if strings.ToLower(opt) == v || strings.HasPrefix(strings.ToLower(opt), v) {
				return i, nil
			}
		}
		// Try as a number
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		return 0, fmt.Errorf("invalid %s value %q; expected one of: %s", envVar, v, strings.Join(options, ", "))
	}
	if !isInteractive() {
		return 0, fmt.Errorf("non-interactive: set %s environment variable", envVar)
	}
	fmt.Println(prompt + ":")
	for i, opt := range options {
		fmt.Printf("  %d. %s\n", i+1, opt)
	}
	for {
		fmt.Printf("Select (1-%d): ", len(options))
		scanner().Scan()
		input := strings.TrimSpace(scanner().Text())
		n, err := strconv.Atoi(input)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		fmt.Println("Invalid selection, try again.")
	}
}

func resolveConfirm(envVar, prompt string) (bool, error) {
	if v := os.Getenv(envVar); v != "" {
		v = strings.ToLower(v)
		return v == "yes" || v == "y" || v == "true" || v == "1", nil
	}
	if !isInteractive() {
		return false, fmt.Errorf("non-interactive: set %s environment variable", envVar)
	}
	fmt.Printf("%s (y/n): ", prompt)
	scanner().Scan()
	input := strings.ToLower(strings.TrimSpace(scanner().Text()))
	return input == "y" || input == "yes", nil
}

type parsedDate struct {
	api     string // YYYY-MM-DD
	display string // January 2, 2026
	short   string // 1/2
}

var dateRegexFull = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{4}$`)
var dateRegexShort = regexp.MustCompile(`^\d{1,2}/\d{1,2}$`)

func resolveDate(envVar string) (parsedDate, error) {
	raw := os.Getenv(envVar)
	if raw == "" {
		if !isInteractive() {
			return parsedDate{}, fmt.Errorf("non-interactive: set %s environment variable", envVar)
		}
		for {
			fmt.Print("Enter event date (MM/DD/YYYY or MM/DD): ")
			scanner().Scan()
			raw = strings.TrimSpace(scanner().Text())
			d, err := parseDate(raw)
			if err != nil {
				fmt.Println(err)
				continue
			}
			return d, nil
		}
	}
	return parseDate(raw)
}

func parseDate(raw string) (parsedDate, error) {
	if dateRegexShort.MatchString(raw) {
		parts := strings.Split(raw, "/")
		month, _ := strconv.Atoi(parts[0])
		day, _ := strconv.Atoi(parts[1])
		year := time.Now().Year()
		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		if t.Month() != time.Month(month) || t.Day() != day {
			return parsedDate{}, fmt.Errorf("invalid date: %s", raw)
		}
		return parsedDate{
			api:     fmt.Sprintf("%d-%02d-%02d", year, month, day),
			display: t.Format("January 2, 2006"),
			short:   fmt.Sprintf("%d/%d", month, day),
		}, nil
	}
	if dateRegexFull.MatchString(raw) {
		parts := strings.Split(raw, "/")
		month, _ := strconv.Atoi(parts[0])
		day, _ := strconv.Atoi(parts[1])
		year, _ := strconv.Atoi(parts[2])
		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		if t.Month() != time.Month(month) || t.Day() != day {
			return parsedDate{}, fmt.Errorf("invalid date: %s", raw)
		}
		return parsedDate{
			api:     fmt.Sprintf("%d-%02d-%02d", year, month, day),
			display: t.Format("January 2, 2006"),
			short:   fmt.Sprintf("%d/%d", month, day),
		}, nil
	}
	return parsedDate{}, fmt.Errorf("invalid date format %q; use MM/DD/YYYY or MM/DD", raw)
}

// --- Event collectors ---

func collectBeavertonHappyHour() (eventEntry, *shift2bikesPayload, error) {
	d, err := resolveDate("EVENT_DATE")
	if err != nil {
		return eventEntry{}, nil, err
	}

	entry := eventEntry{
		Title: fmt.Sprintf("%s Bike Happy Hour", d.short),
		Date:  d.display,
		Start: "Beaverton",
		End:   "Beaverton",
	}

	payload := &shift2bikesPayload{
		Title:         fmt.Sprintf("Westside Bike Happy Hour %s", d.short),
		Details:       "Join us on the westside for Bike Happy Hour. Meet new friends and old, hang out, grab a beverage (alcoholic or not), grab some food, and let's talk bikes! \r\n\r\nEveryone welcome!\r\n\r\nEvery 2nd and 4th Monday, 4:30 to 7 p.m.",
		Audience:      "G",
		Time:          "16:30:00",
		TimeDetails:   "4:30 to 7pm",
		Area:          "W",
		Venue:         "BGs Food Cartel",
		Address:       "4250 SW Rose Biggi Ave Beaverton, OR",
		LocDetails:    "Meet in the back by the bar or in the indoor seating",
		Length:        "--",
		Organizer:     "Ride Westside",
		Email:         "ridewestside2023@gmail.com",
		HideEmail:     "1",
		WebName:       "Ride Westside",
		WebURL:        "https://ridewestside.org",
		CodeOfConduct: "1",
		ReadComic:     "1",
		DateStatuses: []dateStatus{
			{Date: d.api, Status: "A"},
		},
	}

	return entry, payload, nil
}

func collectTigardHappyHour() (eventEntry, *shift2bikesPayload, error) {
	d, err := resolveDate("EVENT_DATE")
	if err != nil {
		return eventEntry{}, nil, err
	}

	entry := eventEntry{
		Title: fmt.Sprintf("%s Tigard Happy Hour", d.short),
		Date:  d.display,
		Start: "Tigard",
		End:   "Tigard",
	}

	// Tigard events don't use Shift2Bikes by default
	return entry, nil, nil
}

func collectCustomEvent() (eventEntry, *shift2bikesPayload, error) {
	d, err := resolveDate("EVENT_DATE")
	if err != nil {
		return eventEntry{}, nil, err
	}

	title, err := resolveValue("EVENT_TITLE", "Event title", "")
	if err != nil {
		return eventEntry{}, nil, err
	}
	details, err := resolveValue("EVENT_DETAILS", "Event description", "")
	if err != nil {
		return eventEntry{}, nil, err
	}
	eventTime, err := resolveValue("EVENT_TIME", "Start time (HH:MM:SS, 24h)", "10:00:00")
	if err != nil {
		return eventEntry{}, nil, err
	}
	timeDetails, err := resolveValue("EVENT_TIME_DETAILS", "Time description (e.g. '10am to 2pm')", "")
	if err != nil {
		return eventEntry{}, nil, err
	}
	venue, err := resolveValue("EVENT_VENUE", "Venue name", "Beaverton Central MAX Station")
	if err != nil {
		return eventEntry{}, nil, err
	}
	address, err := resolveValue("EVENT_ADDRESS", "Address", "12700 SW Crescent St, Beaverton, OR 97005")
	if err != nil {
		return eventEntry{}, nil, err
	}
	area, err := resolveValue("EVENT_AREA", "Area code (N, NE, NW, SE, SW, E, W)", "W")
	if err != nil {
		return eventEntry{}, nil, err
	}
	locDetails, err := resolveOptional("EVENT_LOC_DETAILS", "Location details (optional)")
	if err != nil {
		return eventEntry{}, nil, err
	}
	start, err := resolveValue("EVENT_START", "Start location (for events.md)", "Beaverton")
	if err != nil {
		return eventEntry{}, nil, err
	}
	end, err := resolveValue("EVENT_END", "End location (for events.md)", "Beaverton")
	if err != nil {
		return eventEntry{}, nil, err
	}

	entry := eventEntry{
		Title: title,
		Date:  d.display,
		Start: start,
		End:   end,
	}

	payload := &shift2bikesPayload{
		Title:         title,
		Details:       details,
		Audience:      "G",
		Time:          eventTime,
		TimeDetails:   timeDetails,
		Area:          area,
		Venue:         venue,
		Address:       address,
		LocDetails:    locDetails,
		Length:        "--",
		Organizer:     "Ride Westside",
		Email:         "ridewestside2023@gmail.com",
		HideEmail:     "1",
		WebName:       "Ride Westside",
		WebURL:        "https://ridewestside.org",
		CodeOfConduct: "1",
		ReadComic:     "1",
		DateStatuses: []dateStatus{
			{Date: d.api, Status: "A"},
		},
	}

	return entry, payload, nil
}

// --- Shift2Bikes integration ---

func resolveShiftMode(defaultMode string) (string, error) {
	modes := []string{"create", "existing", "skip"}
	labels := []string{"Create new Shift2Bikes event", "Use existing Shift2Bikes URL", "Skip (no Shift2Bikes link)"}

	if v := os.Getenv("EVENT_SHIFT_MODE"); v != "" {
		v = strings.ToLower(v)
		for _, m := range modes {
			if m == v {
				return m, nil
			}
		}
		return "", fmt.Errorf("invalid EVENT_SHIFT_MODE %q; use create, existing, or skip", v)
	}
	if !isInteractive() {
		if defaultMode != "" {
			return defaultMode, nil
		}
		return "", fmt.Errorf("non-interactive: set EVENT_SHIFT_MODE environment variable")
	}
	idx, err := resolveChoice("", "Shift2Bikes integration", labels)
	if err != nil {
		return "", err
	}
	return modes[idx], nil
}

func handleShift2Bikes(payload *shift2bikesPayload, typeIdx int) (string, error) {
	// Default for Tigard: skip unless explicitly set
	defaultMode := ""
	if typeIdx == 1 {
		defaultMode = "skip"
		if isInteractive() {
			fmt.Println("Tigard events typically don't use Shift2Bikes.")
		}
	}

	mode, err := resolveShiftMode(defaultMode)
	if err != nil {
		return "", err
	}
	return handleShiftMode(mode, payload)
}

func handleShiftMode(mode string, payload *shift2bikesPayload) (string, error) {
	switch strings.ToLower(mode) {
	case "create":
		if payload == nil {
			return "", fmt.Errorf("no Shift2Bikes payload available for this event type")
		}
		url, err := submitToShift2Bikes(payload)
		if err != nil {
			return "", fmt.Errorf("Shift2Bikes API error: %w", err)
		}
		fmt.Println("\nREMINDER: Check the Ride Westside Gmail for the confirmation email")
		fmt.Println("and click the link to publish the event on the Shift2Bikes calendar!")
		return url, nil
	case "existing":
		url, err := resolveValue("EVENT_SHIFT_URL", "Existing Shift2Bikes calendar URL", "")
		if err != nil {
			return "", err
		}
		return url, nil
	case "skip":
		return "", nil
	default:
		return "", fmt.Errorf("invalid EVENT_SHIFT_MODE %q; use create, existing, or skip", mode)
	}
}

func submitToShift2Bikes(payload *shift2bikesPayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://www.shift2bikes.org/api/manage_event.php", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	fmt.Printf("Shift2Bikes response: %s\n", string(respBody))

	var result shift2bikesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.DateStatuses) == 0 || result.DateStatuses[0].ID == "" {
		return "", fmt.Errorf("no event ID in API response")
	}

	eventID := result.DateStatuses[0].ID
	return fmt.Sprintf("https://www.shift2bikes.org/calendar/event-%s", eventID), nil
}

// --- events.md manipulation ---

func formatEventYAML(entry eventEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  - title: %q\n", entry.Title)
	fmt.Fprintf(&b, "    date: %q\n", entry.Date)
	if entry.URL != "" {
		fmt.Fprintf(&b, "    url: %q\n", entry.URL)
	}
	if entry.Route != "" {
		fmt.Fprintf(&b, "    route: %q\n", entry.Route)
	}
	fmt.Fprintf(&b, "    start: %q\n", entry.Start)
	fmt.Fprintf(&b, "    end: %q", entry.End)
	return b.String()
}

func appendEventToFile(path string, entry eventEntry, sectionComment string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	newBlock := "\n" + formatEventYAML(entry)
	insertIdx := -1

	if sectionComment != "" {
		// Find the section comment, then scan forward to the last event in that section
		sectionFound := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, sectionComment) {
				sectionFound = true
				// Scan forward from here to find the last event entry before next section or end
				lastEventLine := i
				for j := i + 1; j < len(lines); j++ {
					jTrimmed := strings.TrimSpace(lines[j])
					// Stop at next comment section or closing ---
					if strings.HasPrefix(jTrimmed, "#") || jTrimmed == "---" {
						break
					}
					// Track lines that are part of event entries
					if jTrimmed != "" {
						lastEventLine = j
					}
				}
				insertIdx = lastEventLine + 1
				break
			}
		}
		if !sectionFound {
			return fmt.Errorf("section comment %q not found in %s", sectionComment, path)
		}
	} else {
		// Insert before closing ---
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) == "---" {
				insertIdx = i
				break
			}
		}
		if insertIdx == -1 {
			return fmt.Errorf("could not find closing --- in %s", path)
		}
	}

	// Splice in the new block
	newLines := make([]string, 0, len(lines)+4)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, newBlock)
	newLines = append(newLines, lines[insertIdx:]...)

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

func determineSection(typeIdx int) (string, error) {
	switch typeIdx {
	case 0:
		return "# Beaverton Bike Happy Hours", nil
	case 1:
		return "# Tigard Happy Hours", nil
	default:
		section, err := resolveOptional("EVENT_SECTION", "Insert after YAML comment section (optional, press enter to append at end)")
		if err != nil {
			return "", err
		}
		if section != "" && !strings.HasPrefix(section, "#") {
			section = "# " + section
		}
		return section, nil
	}
}

// --- Output helpers ---

func printEventSummary(entry eventEntry) {
	fmt.Println("\n===== EVENT SUMMARY =====")
	fmt.Printf("Title: %s\n", entry.Title)
	fmt.Printf("Date:  %s\n", entry.Date)
	if entry.URL != "" {
		fmt.Printf("URL:   %s\n", entry.URL)
	}
	if entry.Route != "" {
		fmt.Printf("Route: %s\n", entry.Route)
	}
	fmt.Printf("Start: %s\n", entry.Start)
	fmt.Printf("End:   %s\n", entry.End)
	fmt.Println("=========================")
}

func printSocialTemplates(entry eventEntry) {
	fmt.Println("\n===== INSTAGRAM POST TEMPLATE =====")
	fmt.Printf("Join us for %s on %s!\n", entry.Title, entry.Date)
	if entry.URL != "" {
		fmt.Printf("Details: %s\n", entry.URL)
	}
	fmt.Println("#BikeHappyHour #RideWestside #BikeLife #Beaverton")
	fmt.Println("===================================")

	fmt.Println("\n===== BLUESKY POST TEMPLATE =====")
	fmt.Printf("%s - %s\n", entry.Title, entry.Date)
	if entry.URL != "" {
		fmt.Printf("Details and directions: %s\n", entry.URL)
	}
	fmt.Println("#BikeHappyHour #RideWestside")
	fmt.Println("=================================")
}
