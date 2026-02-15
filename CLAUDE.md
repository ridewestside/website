# CLAUDE.md

## Project Overview

Hugo static site for Ride Westside, a Portland westside cycling community. Deployed to GitHub Pages at https://beta.ridewestside.org/.

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

  # Tigard Happy Hours - 1st and 3rd Tuesdays
  - title: "1/6 Tigard Happy Hour"
    date: "January 6, 2026"
    start: "Tigard"
    end: "Tigard"
```

Fields: `title` (required), `date`, `url` (optional), `route` (optional), `start`, `end`. Tigard events typically have no Shift2Bikes URL.

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
