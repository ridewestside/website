Add or enrich events in `content/events.md` from a natural-language description, matched against the Shift2Bikes iCal feed.

## Input

$ARGUMENTS

## Process

### 1. Parse the input

Extract each distinct event name and its associated dates. Group multiple dates under one event name when they share the same type (e.g. "Kidical Mass rides on June 13th and 27th" = two instances of one event).

### 2. Search Shift2Bikes for each event type

For each distinct event name, pick 1–2 distinctive keywords and run:

```bash
python3 scripts/ical_events.py "webcal://www.shift2bikes.org/cal/shift-calendar.php" \
  -k "<keyword>" [--start YYYY-MM-DD --end YYYY-MM-DD]
```

Set the date range to cover all requested dates plus a one-month buffer on each end. Run searches in parallel when there are multiple event types.

### 3. Match results to requested dates

For each requested date, find the iCal event whose DTSTART matches. If multiple events match the same date:
- Skip any with a `CANCELLED:` prefix in the summary.
- Prefer the one whose summary best matches the input description.

If no match is found for a date, note it and continue — do not fabricate data.

### 4. Check events.md for existing entries

Before inserting, scan `content/events.md` for an existing entry matching the same event and date (by title date prefix or `url`).

- **Not found** — create a new entry (see step 5).
- **Found and complete** — skip it, note it for the user.
- **Found but missing fields** — update it in place with any fields the Shift2Bikes match can supply (e.g. add `url`, `start_address`, or missing tags). Preserve all fields that are already correct.

### 5. Build YAML entries for new events

For each matched event that needs a new entry, construct a YAML block using this field mapping:

| events.md field | Source |
|---|---|
| `title` | iCal SUMMARY, stripped of leading `"Ride Westside - "` or `"Ride_WestSide - "`. Prepend `M/D ` date prefix (e.g. `6/13`). |
| `date` | Full month-name date string: `"June 13, 2026"` |
| `url` | iCal URL field |
| `start` | City/neighborhood extracted from iCal LOCATION. Common westside values: `"Beaverton"`, `"Hillsboro"`, `"Tigard"`, `"Quatama"`, `"Sunset TC"`, `"Orenco"`, `"Cedar Park"`. |
| `end` | Same as `start` for round trips. Use destination city for one-way rides (e.g. r2r events). |
| `start_address` | Street address from iCal LOCATION, formatted as `"123 Main St, City, OR Zip"`. If location has coordinates only (e.g. Quatama MAX), use the known intersection: `"NW 205th Ave & NW Quatama Rd, Hillsboro, OR 97124"`. Omit if no address is available. |
| `tags` | Infer from summary and description — see tag rules below |

**Tag inference** — read the iCal description carefully; do not default to any tag:
- `ride` — it's a group bike ride
- `happy-hour` — social/happy hour gathering (not necessarily a ride)
- `family-friendly` — explicitly welcoming to families and children
- `r2r` — the group rides to attend another event ("ride to ride", "R2R")
- `cause` — charity, fundraiser, or support ride
- `challenging` — long distance, significant elevation, or description signals difficulty
- `festival` — cycling festival or large community event
- `not-rws` — contact field does not mention Ride Westside / WitchyWillow / Ride_WestSide

Multiple tags are allowed. An event can have no tags if none apply clearly.

### 6. Insert into the correct section

- Happy hours → `# Beaverton Bike Happy Hours` or `# Tigard Happy Hours`
- Festivals and large community events → `# Festivals`
- Everything else → `# Rides`

Append new entries at the end of the appropriate section. Run `mage build` to verify the YAML is valid.

### 7. Report and commit

Tell the user: what was added, what was updated, and what couldn't be matched. Then commit with a concise message summarising the changes.
