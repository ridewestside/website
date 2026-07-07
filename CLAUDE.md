# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Hugo static site for Ride Westside, a Portland westside cycling community focused on rides originating from the Beaverton / Tigard / Hillsboro area. Deployed to GitHub Pages at https://beta.ridewestside.org/.

The site is a single-page link hub: social icons, an About section, a filterable upcoming events list, and a press section. All dynamic behavior (event sorting, filtering, map buttons, share buttons) runs client-side via compiled TypeScript. There is no backend.

### How Events Appear on the Frontend

JavaScript reads each event card's `data-date` attribute on page load and sorts events into three sections:

- **Upcoming Rides** — events from today through the next 90 days, sorted soonest first
- **Later This Year** — events more than 90 days out, collapsed by default
- **Past Rides** — events before today, sorted most-recent first, collapsed by default. Cards generated from a `recurring` rule (`data-recurring`) are the exception — once past, they're removed from the DOM entirely rather than added here, so routine happy hours don't clutter the historical record. See Recurring Events below.

Each card renders:
- Title and date
- Location display (`Start` or `Start → End` if different)
- Tag chips (clickable — clicking a chip sets that tag as the active filter)
- **View Event** button (requires `url`)
- **Route** button (requires `route`)
- **Navigate** button (requires `start_address`) — opens address in map app
- **Share** button — native share sheet or clipboard fallback

Filters (Start, End, Tag) persist across page loads via `localStorage` and URL query params (`?start=Beaverton&tag=ride`).

## Build & Dev Commands

```bash
mage build          # Build TypeScript + Hugo (default target)
mage serve          # Dev server with hot reload (builds TS first)
mage dev            # Alias for serve
mage watch          # Watch TypeScript for changes (run in separate terminal)
mage buildTS        # Compile TypeScript only
mage clean          # Remove public/ directory
mage checkLinks         # Validate all external links in built site
mage checkProse         # Vale prose linting (bot-speak, hedge phrases, word substitutions)
mage validateContent    # Validate project page front matter (required fields, valid status)
mage checkInternalLinks # Validate all internal links in built site resolve to real pages
```

### Event Management

```bash
mage addEvent                    # Interactive event creation wizard (one-off events)
mage generateRecurringEvents     # Expand `recurring` rules into data/generated_events.yaml (also runs as part of build/serve/dev)
```

`addEvent` supports non-interactive use via environment variables (EVENT_TYPE, EVENT_DATE, EVENT_SHIFT_MODE, etc.). In a non-interactive terminal, missing required vars cause exit 1. It's for one-off events; recurring series are declared in `events.md`'s `recurring` list (see below), not added one occurrence at a time.

## Project Structure

- `content/events.md` — Event database: hand-authored one-off events (YAML front matter, section comments) plus `recurring` rules for series like happy hours
- `internal/recurring/` — Shared Go package that expands a `recurring` rule into concrete dated events; used by both the Hugo data generator and `cmd/buildpdf`, so they always agree on what's scheduled
- `src/main.ts` — Client-side TypeScript (event filtering, collapsible sections, share buttons, dropping stale recurring cards)
- `themes/linkpage/` — Custom Hugo theme (layouts, CSS, compiled JS)
  - `layouts/partials/event-card.html` — Shared card markup, used for both hand-authored and generated events
- `magefiles/` — Mage build tasks (Go)
  - `magefile.go` — Build, serve, clean targets
  - `checklinks.go` — Link validation with Shift2Bikes API awareness
  - `addevent.go` — One-off event creation wizard
  - `recurring.go` — `GenerateRecurringEvents`, the build step that writes `data/generated_events.yaml`
  - `buildpdf.go` — Generates `public/events.pdf` via `cmd/buildpdf`
- `data/generated_events.yaml` — Build-generated, gitignored; Hugo data file produced by `mage generateRecurringEvents`, consumed by `layouts/index.html` and `layouts/index.ics`
- `scripts/ical_events.py` — Fetch and filter events from an iCal file or URL (see below)
- `.github/workflows/` — CI/CD + event management workflows
- `.claude/commands/` — Project-specific Claude Code slash commands

## Key Patterns

### events.md Format

One-off events are YAML list items under `events:`, organized by section comments:
```yaml
events:
  # Rides
  - title: "5/17 Pittock Mansion Ride"
    date: "May 17, 2026"
    url: "https://shift2bikes.org/calendar/event-12345"
    start: "Beaverton"
    end: "Portland"
    start_address: "4250 SW Rose Biggi Ave, Beaverton, OR"
    tags: [ride, challenging]
```

Recurring series (happy hours, etc.) go in the separate `recurring:` list instead — see the Recurring Events section below.

### Event Fields

| Field | Required | Description |
|-------|----------|-------------|
| `title` | Yes | Event name. Convention: `M/D EventName` prefix (e.g. `5/17 Pittock Mansion Ride`) |
| `date` | Yes | Display date string: `"January 12, 2026"`. Used for sorting events into Upcoming / Later This Year / Past sections |
| `url` | No | Shift2Bikes calendar URL. Renders a "View Event" button. Omit for events without a Shift2Bikes listing |
| `route` | No | RideWithGPS route URL. Renders a "Route" button on the card |
| `start` | No | Starting location name (e.g. `"Beaverton"`, `"Tigard"`, `"Quatama"`). Populates the Start filter dropdown and the location display on the card |
| `end` | No | Ending location name. Populates the End filter dropdown. If equal to `start`, shown once; if different, shown as `Start → End` |
| `start_address` | No | Full street address of the start location (e.g. `"12725 SW Millikan Way, Beaverton, OR 97005"`). When present, adds a navigation button that opens the address in the user's map app (Apple Maps, Google Maps, or OpenStreetMap based on device/settings) |
| `tags` | No | YAML list of tag strings. Rendered as clickable filter chips on the card. See tag reference below |

### Tags

Tags appear as clickable chips on each event card and populate the Tag filter dropdown. Multiple tags are supported.

| Tag | Meaning |
|-----|---------|
| `happy-hour` | Bike Happy Hour social event |
| `ride` | Group ride |
| `r2r` | Ride to Ride — the group rides to attend another event |
| `not-rws` | Not a Ride Westside event; listed for community awareness |
| `cause` | Charity or cause-related ride |
| `festival` | Cycling festival or community event |
| `challenging` | Ride with significant elevation, technical terrain, or above-average difficulty |
| `family-friendly` | Explicitly welcoming to families and children |

New tags can be added freely — the frontend discovers all tags at runtime and populates the filter dropdown automatically.

### events.md Sections

Hand-authored, one-off events under `events:` are grouped by YAML comment headers. Recurring series (Beaverton, Tigard, Tanasbourne happy hours) live under the separate `recurring` list instead — see below. Sections currently in `events:`:

- `# Hillsboro Bike Happy Hours (not RWS) - 1st and 3rd Tuesdays` — organized independently, not by Ride Westside; add future occurrences manually as Shift2Bikes links become available (not a `recurring` rule, since future dates/links aren't known ahead of time)
- `# Festivals` — community events, festivals, not-RWS rides
- `# Activism & Engagement` — advocacy meetings and milestones
- `# Rides` — all other rides; append new rides here

### File Manipulation

events.md is edited line-by-line (not YAML-parsed) to preserve comments and formatting. New events are inserted at the end of their section, identified by the `#` comment header.

## Recurring Events

Recurring series (happy hours, etc.) are declared once as a rule in the `recurring` list in `content/events.md`'s front matter, instead of as dozens of individually dated `events` entries. `mage generateRecurringEvents` (a dependency of `build`/`serve`/`dev`) expands every rule into concrete dated events and writes them to `data/generated_events.yaml`, a gitignored Hugo data file rendered by `layouts/index.html` and `layouts/index.ics` alongside the hand-authored `events` list, using the same card partial.

A rule never generates an occurrence before today, so already-past instances of a series simply aren't produced — there's nothing to prune. It generates forward through `until` (if set) or `horizon_days` (default 365) otherwise.

```yaml
recurring:
  - title: "Bike Happy Hour"        # occurrence titles become "M/D Bike Happy Hour"
    weekday: monday                 # sunday .. saturday
    nth: [2, 4]                     # which occurrence(s) of that weekday in the month; a missing nth (e.g. "5th Monday" in a 4-Monday month) is silently skipped
    start: "Beaverton"
    end: "Beaverton"
    start_address: "4250 SW Rose Biggi Ave, Beaverton, OR"
    tags: [happy-hour]
    url: "https://shift2bikes.org/..."   # optional default URL for every occurrence (e.g. one shared Shift2Bikes recurring-series link)
    since: "2026-06-01"              # optional; don't generate before this date
    until: "2026-08-31"              # optional; stop generating after this date (omit for an indefinite series)
    horizon_days: 365                # optional; overrides the 365-day default generation window
    overrides:                       # optional per-date exceptions, keyed "YYYY-MM-DD"
      "2026-07-13": { url: "https://shift2bikes.org/calendar/event-23104" }
```

Use `overrides` when each occurrence gets its own confirmed Shift2Bikes URL (Beaverton, Tigard); use the rule-level `url` when the whole series shares one link (Tanasbourne). A generated occurrence with no URL just omits the "View Event" button's target until one is added.

**Staleness caveat**: the site only rebuilds on push to `main` (no scheduled rebuild), so `data/generated_events.yaml`'s generation window and past/upcoming cutoff are only as fresh as the last build. To guarantee a recurring occurrence never lingers in "Past Rides" between builds, `src/main.ts` removes any card with `data-recurring` from the DOM entirely once its date has passed, rather than moving it to Past Rides — so visitors never see stale recurring clutter regardless of rebuild cadence. New occurrences beyond the last build's 365-day window won't appear until the next build, though.

- **Beaverton Happy Hour + Post Ride**: 2nd and 4th Monday of each month
- **Tigard Happy Hour**: 2nd and 4th Thursday of each month (Cooper Mountain Ale Works, Tigard)
- **Tanasbourne Bike Happy Hour + Post Ride** (Prime Tap House, 1896 NE 106th Ave, Beaverton): 1st, 3rd, and 5th (if applicable) Monday, 4:30–7:00 PM Happy Hour followed by a 10–20 mile ride at 7:00 PM. Currently a June–August 2026 trial (`since`/`until` bound it); all occurrences share one Shift2Bikes event link.

## Tool Versions

Managed via `mise.toml`: Go 1.25.5, Node 22, mage (latest), esbuild (latest), TypeScript (latest).

## Build Pipeline

TypeScript → esbuild (bundle/minify/sourcemap) → `themes/linkpage/static/js/main.js`, plus `recurring` rules → `mage generateRecurringEvents` → `data/generated_events.yaml` → Hugo (--gc --minify) → `public/`

## scripts/ical_events.py

Fetches and filters events from the Shift2Bikes iCal feed (or any `.ics` file/URL). Useful for enriching manually-described events with Shift2Bikes URLs, addresses, and metadata before adding them to `events.md`.

```bash
# Requires: pip install recurring-ical-events icalendar
python3 scripts/ical_events.py "webcal://www.shift2bikes.org/cal/shift-calendar.php" \
  -k "keyword" --start 2026-06-01 --end 2026-09-01
```

- `-k WORD` — keyword filter (repeatable; all must match); searches summary, contact, description, location
- `-f FIELD` — restrict search to a specific field (`summary`, `contact`, `description`, `location`, `all`)
- `--start` / `--end` — date range (default: today + 1 year)
- `--count` — print count only
- `-v` — show cache status (responses are cached via ETag/Last-Modified at `~/.cache/ical_events/`)

Output includes event title, date/time, location, contact, Shift2Bikes URL, and truncated description.

## GitHub Workflows

- **hugo.yml** — Deploy to Pages on push to `main`
- **add-event.yml** — Manual form to add a single one-off event (creates PR)

Event workflows create PRs (not direct pushes) for review before publishing. There's no equivalent workflow for recurring series — edit the `recurring` rule in `content/events.md` directly (e.g. to add `overrides` once a Shift2Bikes URL is confirmed, or to set `until` when a series ends).

## Deployment

- **GitHub Pages** — primary deployment, triggered on push to `main` via `hugo.yml`
- **Netlify** — deploy previews for PRs; `netlify.toml` sets build command and Hugo version

## External Integrations

- **Shift2Bikes iCal** (`webcal://www.shift2bikes.org/cal/shift-calendar.php`) — community calendar; use `scripts/ical_events.py` to query it
- **Shift2Bikes API** (`https://www.shift2bikes.org/api/manage_event.php`) — event creation; requires email confirmation after API submission
- **Shift2Bikes event validation** — `checkLinks` validates event URLs via API at `shift2bikes.org/api/events.php?id=` instead of scraping the SPA
