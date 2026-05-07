#!/usr/bin/env python3
"""Extract and filter events from an iCal file or URL."""

import argparse
import hashlib
import json
import sys
import urllib.request
from datetime import date, datetime, timezone
from pathlib import Path

import recurring_ical_events
from icalendar import Calendar

FILTER_FIELDS = ("summary", "contact", "description", "location")
CACHE_DIR = Path.home() / ".cache" / "ical_events"


def _cache_paths(url: str) -> tuple[Path, Path]:
    key = hashlib.sha256(url.encode()).hexdigest()[:16]
    return CACHE_DIR / f"{key}.ics", CACHE_DIR / f"{key}.json"


def load_calendar(source: str, *, verbose: bool = False) -> Calendar:
    if source.startswith("http://") or source.startswith("webcal://") or source.startswith("https://"):
        data = _fetch_url(source, verbose=verbose)
    else:
        with open(source, "rb") as f:
            data = f.read()
    return Calendar.from_ical(data)


def _fetch_url(source: str, *, verbose: bool = False) -> bytes:
    url = source.replace("webcal://", "https://")
    cache_ics, cache_meta = _cache_paths(url)

    etag = last_modified = None
    if cache_meta.exists():
        meta = json.loads(cache_meta.read_text())
        etag = meta.get("etag")
        last_modified = meta.get("last_modified")

    req = urllib.request.Request(url)
    if etag:
        req.add_header("If-None-Match", etag)
    if last_modified:
        req.add_header("If-Modified-Since", last_modified)

    try:
        with urllib.request.urlopen(req) as resp:
            data = resp.read()
            new_etag = resp.headers.get("ETag")
            new_lm = resp.headers.get("Last-Modified")
        if verbose:
            print(f"[cache] downloaded {len(data):,} bytes from {url}", file=sys.stderr)
        CACHE_DIR.mkdir(parents=True, exist_ok=True)
        cache_ics.write_bytes(data)
        cache_meta.write_text(json.dumps({"etag": new_etag, "last_modified": new_lm}))
        return data
    except urllib.error.HTTPError as e:
        if e.code == 304 and cache_ics.exists():
            if verbose:
                print(f"[cache] 304 Not Modified — using cached copy", file=sys.stderr)
            return cache_ics.read_bytes()
        raise


def get_field(event, field: str) -> str:
    value = event.get(field.upper())
    if value is None:
        return ""
    return str(value)


def event_matches(event, keywords: list[str], fields: list[str]) -> bool:
    if not keywords:
        return True
    haystack = " ".join(get_field(event, f) for f in fields).lower()
    return all(kw.lower() in haystack for kw in keywords)


def format_event(event) -> str:
    summary = get_field(event, "summary") or "(no title)"
    dtstart = event.get("DTSTART")
    if dtstart:
        dt = dtstart.dt
        if isinstance(dt, datetime):
            local = dt.astimezone().strftime("%Y-%m-%d %H:%M %Z")
        else:
            local = str(dt)
    else:
        local = "unknown date"
    location = get_field(event, "location").replace("\n", ", ").strip()
    contact = get_field(event, "contact")
    description = get_field(event, "description")
    url = get_field(event, "url")
    lines = [f"  {summary}  [{local}]"]
    if location:
        lines.append(f"    Location:    {location}")
    if contact:
        lines.append(f"    Contact:     {contact}")
    if url:
        lines.append(f"    URL:         {url}")
    if description:
        short = description.replace("\\n", " ").replace("\n", " ").strip()
        if len(short) > 200:
            short = short[:197] + "..."
        lines.append(f"    Description: {short}")
    return "\n".join(lines)


def parse_date(s: str) -> date:
    return datetime.strptime(s, "%Y-%m-%d").date()


def main():
    parser = argparse.ArgumentParser(
        description="Extract events from an iCal file or URL, with optional keyword filtering."
    )
    parser.add_argument("source", help="Path to .ics file or http(s)/webcal URL")
    parser.add_argument(
        "--start",
        metavar="YYYY-MM-DD",
        help="Start of date range (default: today)",
    )
    parser.add_argument(
        "--end",
        metavar="YYYY-MM-DD",
        help="End of date range (default: one year from start)",
    )
    parser.add_argument(
        "--keyword",
        "-k",
        metavar="WORD",
        action="append",
        dest="keywords",
        default=[],
        help="Keyword to match (repeatable; all must match)",
    )
    parser.add_argument(
        "--field",
        "-f",
        metavar="FIELD",
        action="append",
        dest="fields",
        choices=list(FILTER_FIELDS) + ["all"],
        default=[],
        help=f"Field(s) to search: {', '.join(FILTER_FIELDS)}, all (default: all)",
    )
    parser.add_argument(
        "--count",
        action="store_true",
        help="Print only the count of matching events",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Print cache/fetch status to stderr",
    )
    args = parser.parse_args()

    start = parse_date(args.start) if args.start else date.today()
    end = parse_date(args.end) if args.end else date(start.year + 1, start.month, start.day)

    fields = args.fields or ["all"]
    if "all" in fields:
        search_fields = list(FILTER_FIELDS)
    else:
        search_fields = args.fields

    try:
        cal = load_calendar(args.source, verbose=args.verbose)
    except Exception as e:
        print(f"Error loading calendar: {e}", file=sys.stderr)
        sys.exit(1)

    events = recurring_ical_events.of(cal).between(start, end)
    events.sort(key=lambda e: e["DTSTART"].dt if e.get("DTSTART") else date.min)

    matched = [e for e in events if event_matches(e, args.keywords, search_fields)]

    if args.count:
        print(len(matched))
        return

    if not matched:
        print("No matching events found.")
        return

    field_desc = "all fields" if "all" in (args.fields or ["all"]) else ", ".join(search_fields)
    kw_desc = f" matching {args.keywords!r} in {field_desc}" if args.keywords else ""
    print(f"{len(matched)} event(s) from {start} to {end}{kw_desc}:\n")
    for event in matched:
        print(format_event(event))
        print()


if __name__ == "__main__":
    main()
