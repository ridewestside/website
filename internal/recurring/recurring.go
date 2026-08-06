// Package recurring expands declarative recurring-event rules (defined in
// content/events.md front matter) into concrete dated events. It's shared
// between the mage build step that feeds Hugo's event cards and cmd/buildpdf,
// so both always agree on what's currently scheduled.
package recurring

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Event is a single concrete event card, whether hand-authored in the
// `events` list or produced by expanding a Rule.
type Event struct {
	Title        string   `yaml:"title"`
	Date         string   `yaml:"date"`
	URL          string   `yaml:"url,omitempty"`
	Route        string   `yaml:"route,omitempty"`
	Start        string   `yaml:"start,omitempty"`
	End          string   `yaml:"end,omitempty"`
	StartAddress string   `yaml:"start_address,omitempty"`
	StartTime    string   `yaml:"start_time,omitempty"`
	EndTime      string   `yaml:"end_time,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
	Recurring    bool     `yaml:"recurring,omitempty"`
}

// Override supplies per-occurrence data (typically a confirmed Shift2Bikes
// URL) for a rule-generated date, keyed by that date in "2006-01-02" form.
type Override struct {
	URL   string `yaml:"url,omitempty"`
	Route string `yaml:"route,omitempty"`
}

// Rule describes a recurring event series: which nth weekday(s) of the
// month it falls on, and the window in which it's active. Occurrences are
// only ever generated from today forward (never in the past) through
// Until, or through HorizonDays (defaulting to DefaultHorizonDays) if Until
// is unset.
type Rule struct {
	Title        string              `yaml:"title"`
	Weekday      string              `yaml:"weekday"`
	Nth          []int               `yaml:"nth"`
	Start        string              `yaml:"start,omitempty"`
	End          string              `yaml:"end,omitempty"`
	StartAddress string              `yaml:"start_address,omitempty"`
	StartTime    string              `yaml:"start_time,omitempty"`
	EndTime      string              `yaml:"end_time,omitempty"`
	Tags         []string            `yaml:"tags,omitempty"`
	URL          string              `yaml:"url,omitempty"`
	Since        string              `yaml:"since,omitempty"`
	Until        string              `yaml:"until,omitempty"`
	HorizonDays  int                 `yaml:"horizon_days,omitempty"`
	Overrides    map[string]Override `yaml:"overrides,omitempty"`
}

// FrontMatter is the shape of content/events.md's YAML front matter that
// this package cares about.
type FrontMatter struct {
	Events    []Event `yaml:"events"`
	Recurring []Rule  `yaml:"recurring"`
}

// DefaultHorizonDays is how far into the future a Rule generates occurrences
// when it doesn't set its own HorizonDays.
const DefaultHorizonDays = 365

// LoadFrontMatter parses the YAML front matter of a Hugo content file such
// as content/events.md.
func LoadFrontMatter(path string) (*FrontMatter, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(raw), "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("no YAML front matter in %s", path)
	}
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, fmt.Errorf("parsing front matter in %s: %w", path, err)
	}
	return &fm, nil
}

var weekdays = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// nthWeekdayOfMonth returns the nth occurrence of weekday in the given
// month/year, or the zero Time if that month doesn't have an nth occurrence
// (e.g. a "5th Monday" in a four-Monday month).
func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	offset := int(weekday) - int(first.Weekday())
	if offset < 0 {
		offset += 7
	}
	firstOccurrence := first.AddDate(0, 0, offset)
	target := firstOccurrence.AddDate(0, 0, (n-1)*7)
	if target.Month() != month {
		return time.Time{}
	}
	return target
}

// Expand generates concrete Events for a single rule, from whichever is
// later of "now" or the rule's Since, through the rule's Until (or the
// horizon if Until is unset). It never generates an occurrence before now,
// so already-past instances of a recurring series are simply never
// produced — there's nothing to prune later.
func Expand(rule Rule, now time.Time) ([]Event, error) {
	weekday, ok := weekdays[strings.ToLower(rule.Weekday)]
	if !ok {
		return nil, fmt.Errorf("recurring rule %q: invalid weekday %q", rule.Title, rule.Weekday)
	}
	if len(rule.Nth) == 0 {
		return nil, fmt.Errorf("recurring rule %q: no nth occurrences configured", rule.Title)
	}

	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	windowStart := today
	if rule.Since != "" {
		since, err := time.ParseInLocation("2006-01-02", rule.Since, loc)
		if err != nil {
			return nil, fmt.Errorf("recurring rule %q: invalid since date %q: %w", rule.Title, rule.Since, err)
		}
		if since.After(windowStart) {
			windowStart = since
		}
	}

	horizonDays := rule.HorizonDays
	if horizonDays <= 0 {
		horizonDays = DefaultHorizonDays
	}
	windowEnd := today.AddDate(0, 0, horizonDays)
	if rule.Until != "" {
		until, err := time.ParseInLocation("2006-01-02", rule.Until, loc)
		if err != nil {
			return nil, fmt.Errorf("recurring rule %q: invalid until date %q: %w", rule.Title, rule.Until, err)
		}
		if until.Before(windowEnd) {
			windowEnd = until
		}
	}
	if windowEnd.Before(windowStart) {
		return nil, nil
	}

	type dated struct {
		t time.Time
		e Event
	}
	var found []dated

	cursor := time.Date(windowStart.Year(), windowStart.Month(), 1, 0, 0, 0, 0, loc)
	for !cursor.After(windowEnd) {
		for _, n := range rule.Nth {
			d := nthWeekdayOfMonth(cursor.Year(), cursor.Month(), weekday, n)
			if d.IsZero() || d.Before(windowStart) || d.After(windowEnd) {
				continue
			}
			found = append(found, dated{t: d, e: buildEvent(rule, d)})
		}
		cursor = cursor.AddDate(0, 1, 0)
	}

	sort.Slice(found, func(i, j int) bool { return found[i].t.Before(found[j].t) })

	events := make([]Event, len(found))
	for i, f := range found {
		events[i] = f.e
	}
	return events, nil
}

func buildEvent(rule Rule, d time.Time) Event {
	url := rule.URL
	route := ""
	if o, ok := rule.Overrides[d.Format("2006-01-02")]; ok {
		if o.URL != "" {
			url = o.URL
		}
		route = o.Route
	}
	return Event{
		Title:        fmt.Sprintf("%d/%d %s", int(d.Month()), d.Day(), rule.Title),
		Date:         d.Format("January 2, 2006"),
		URL:          url,
		Route:        route,
		Start:        rule.Start,
		End:          rule.End,
		StartAddress: rule.StartAddress,
		StartTime:    rule.StartTime,
		EndTime:      rule.EndTime,
		Tags:         rule.Tags,
		Recurring:    true,
	}
}

// ExpandAll expands every rule and returns the combined events, in the same
// date order Expand already guarantees per rule (interleaved by rule, not
// globally re-sorted, since callers typically want each rule's timeline
// grouped together, or sort themselves if they need a single timeline).
func ExpandAll(rules []Rule, now time.Time) ([]Event, error) {
	var all []Event
	for _, rule := range rules {
		events, err := Expand(rule, now)
		if err != nil {
			return nil, err
		}
		all = append(all, events...)
	}
	return all, nil
}

// GenerateDataFile reads eventsPath's `recurring` rules, expands them as of
// now, and writes the result to outPath as a Hugo data file. It's the single
// implementation shared by the mage build step and cmd/generate-events, so
// every place that builds the site (local dev, GitHub Actions, Netlify)
// produces the same generated events regardless of which of those entry
// points it uses.
func GenerateDataFile(eventsPath, outPath string, now time.Time) (int, error) {
	fm, err := LoadFrontMatter(eventsPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", eventsPath, err)
	}

	events, err := ExpandAll(fm.Recurring, now)
	if err != nil {
		return 0, err
	}

	if dir := filepath.Dir(outPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, err
		}
	}

	out, err := yaml.Marshal(struct {
		Events []Event `yaml:"events"`
	}{Events: events})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal generated events: %w", err)
	}

	if err := os.WriteFile(outPath, out, 0644); err != nil {
		return 0, fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return len(events), nil
}
