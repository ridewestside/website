//go:build mage

package main

import (
	"fmt"
	"time"

	"github.com/ridewestside/website/internal/recurring"
)

const generatedEventsPath = "data/generated_events.yaml"

// GenerateRecurringEvents expands the `recurring` rules in content/events.md
// into concrete dated events and writes them to data/generated_events.yaml,
// a Hugo data file the events section and calendar feed render alongside the
// hand-authored `events` list. It's a build dependency of Build/Serve/Dev.
//
// cmd/generate-events is the same logic packaged as a plain Go binary for
// CI pipelines (GitHub Actions, Netlify) that build the site without mage —
// keep both entry points wired up when the pipeline changes.
//
// The generated window (which occurrences exist, and the cutoff between
// past/upcoming) is only as fresh as the last time this ran — see the
// "Recurring Events" section in CLAUDE.md.
func GenerateRecurringEvents() error {
	n, err := recurring.GenerateDataFile("content/events.md", generatedEventsPath, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Generated %d recurring event occurrence(s) into %s\n", n, generatedEventsPath)
	return nil
}
