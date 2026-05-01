# CLAUDE.md

## Project Overview

Hugo static site for Ride Westside, a Portland westside cycling community focused on rides originating from the Beaverton / Tigard / Hillsboro area. Deployed to GitHub Pages at https://beta.ridewestside.org/.

The site is a single-page link hub: social icons, an About section, a filterable upcoming events list, and a press section. All dynamic behavior (event sorting, filtering, map buttons, share buttons) runs client-side via compiled TypeScript. There is no backend.

### How Events Appear on the Frontend

JavaScript reads each event card's `data-date` attribute on page load and sorts events into three sections:

- **Upcoming Rides** — events from today through the next 90 days, sorted soonest first
- **Later This Year** — events more than 90 days out, collapsed by default
- **Past Rides** — events before today, sorted most-recent first, collapsed by default

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
mage checkLinks     # Validate all external links in built site
```

### Event Management

```bash
mage addEvent                    # Interactive event creation wizard
mage addRecurringEvents 2027     # Generate a year of recurring happy hours (idempotent)
```

`addEvent` supports non-interactive use via environment variables (EVENT_TYPE, EVENT_DATE, EVENT_SHIFT_MODE, etc.). In a non-interactive terminal, missing required vars cause exit 1.

## Project Structure

- `content/events.md` — Event database (YAML front matter with section comments)
- `src/main.ts` — Client-side TypeScript (event filtering, collapsible sections, share buttons)
- `themes/linkpage/` — Custom Hugo theme (layouts, CSS, compiled JS)
- `magefiles/` — Mage build tasks (Go)
  - `magefile.go` — Build, serve, clean targets
  - `checklinks.go` — Link validation with Shift2Bikes API awareness
  - `addevent.go` — Event creation (single + recurring)
- `.github/workflows/` — CI/CD + event management workflows

## Key Patterns

### events.md Format

Events are YAML list items in front matter, organized by section comments:
```yaml
events:
  # Beaverton Bike Happy Hours
  - title: "1/12 Bike Happy Hour"
    date: "January 12, 2026"
    url: "https://shift2bikes.org/calendar/event-23092"
    start: "Beaverton"
    end: "Beaverton"
    start_address: "4250 SW Rose Biggi Ave, Beaverton, OR"
    tags: [happy-hour]

  # Tigard Happy Hours - 1st and 3rd Tuesdays
  - title: "1/6 Tigard Happy Hour"
    date: "January 6, 2026"
    start: "Tigard"
    end: "Tigard"
    tags: [happy-hour]
```

### Event Fields

| Field | Required | Description |
|-------|----------|-------------|
| `title` | Yes | Event name. Convention: `M/D EventName` prefix (e.g. `5/17 Pittock Mansion Ride`) |
| `date` | Yes | Display date string: `"January 12, 2026"`. Used for sorting events into Upcoming / Later This Year / Past sections |
| `url` | No | Shift2Bikes calendar URL. Renders a "View Event" button. Omit for Tigard events or events without a Shift2Bikes listing |
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

New tags can be added freely — the frontend discovers all tags at runtime and populates the filter dropdown automatically.

### events.md Sections

Events are grouped by YAML comment headers. The `addEvent` wizard uses these headers to find the right insertion point. Sections in order:

- `# Beaverton Bike Happy Hours` — 2nd and 4th Mondays; managed by `addRecurringEvents`
- `# Tigard Happy Hours - 1st and 3rd Tuesdays` — 1st and 3rd Tuesdays; managed by `addRecurringEvents`
- `# Ride to ride` — rides that travel to another event
- `# Causes` — charity and cause rides
- `# Special Rides` — featured or challenging rides
- `# Memes` — test and joke events
- `# Critical Mass` — (reserved)
- `# Festivals` — festivals and community events

### File Manipulation

events.md is edited line-by-line (not YAML-parsed) to preserve comments and formatting. New events are inserted at the end of their section, identified by the `#` comment header.

### Idempotency

`addRecurringEvents` deduplicates by matching event titles. Running it multiple times for the same year produces no changes.

## Recurring Event Schedule

- **Beaverton Happy Hours**: 2nd and 4th Monday of each month
- **Tigard Happy Hours**: 1st and 3rd Tuesday of each month

## Tool Versions

Managed via `mise.toml`: Go 1.25.5, Node 22, mage (latest), esbuild (latest), TypeScript (latest).

## Build Pipeline

TypeScript → esbuild (bundle/minify/sourcemap) → `themes/linkpage/static/js/main.js` → Hugo (--gc --minify) → `public/`

## GitHub Workflows

- **hugo.yml** — Deploy to Pages on push to `main`
- **add-event.yml** — Manual form to add a single event (creates PR)
- **add-recurring-events.yml** — Manual form to generate a year of events (creates PR)

Event workflows create PRs (not direct pushes) for review before publishing.

## External Integrations

- **Shift2Bikes API** (`https://www.shift2bikes.org/api/manage_event.php`) — Event creation. Requires email confirmation after API submission.
- **Shift2Bikes event validation** — `checkLinks` validates event URLs via API at `shift2bikes.org/api/events.php?id=` instead of scraping the SPA.
