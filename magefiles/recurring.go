//go:build mage

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ridewestside/website/internal/recurring"
	"gopkg.in/yaml.v3"
)

const generatedEventsPath = "data/generated_events.yaml"

// GenerateRecurringEvents expands the `recurring` rules in content/events.md
// into concrete dated events and writes them to data/generated_events.yaml,
// a Hugo data file the events section and calendar feed render alongside the
// hand-authored `events` list. It's a build dependency of Build/Serve/Dev.
//
// The generated window (which occurrences exist, and the cutoff between
// past/upcoming) is only as fresh as the last time this ran — see the
// "Recurring Events" section in CLAUDE.md.
func GenerateRecurringEvents() error {
	fm, err := recurring.LoadFrontMatter("content/events.md")
	if err != nil {
		return fmt.Errorf("failed to read content/events.md: %w", err)
	}

	events, err := recurring.ExpandAll(fm.Recurring, time.Now())
	if err != nil {
		return err
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	out, err := yaml.Marshal(struct {
		Events []recurring.Event `yaml:"events"`
	}{Events: events})
	if err != nil {
		return fmt.Errorf("failed to marshal generated events: %w", err)
	}

	if err := os.WriteFile(generatedEventsPath, out, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", generatedEventsPath, err)
	}

	fmt.Printf("Generated %d recurring event occurrence(s) into %s\n", len(events), generatedEventsPath)
	return nil
}
