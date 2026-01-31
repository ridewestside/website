/**
 * Creates an event on Shift2Bikes and appends it to content/events.md.
 *
 * Environment variables (inputs):
 *   EVENT_TYPE        - "happy-hour" | "custom"
 *   EVENT_DATE        - MM/DD or MM/DD/YYYY
 *   EVENT_TITLE       - (custom only)
 *   EVENT_DETAILS     - (custom only)
 *   EVENT_TIME        - HH:MM:SS 24h (custom only, default 16:30:00)
 *   EVENT_TIME_DETAIL - human-readable time (custom only)
 *   EVENT_AREA        - area code (custom only, default W)
 *   EVENT_VENUE       - venue name (custom only)
 *   EVENT_ADDRESS     - venue address (custom only)
 *   EVENT_LOC_DETAILS - location details (custom only)
 *   EVENT_AUDIENCE    - audience rating (custom only, default G)
 *   EVENT_START       - start location for events.md (default Beaverton)
 *   EVENT_END         - end location for events.md (default Beaverton)
 *   EVENT_ROUTE       - optional RideWithGPS route URL
 */

import * as fs from "node:fs";
import * as path from "node:path";

const SHIFT2BIKES_API = "https://www.shift2bikes.org/api/manage_event.php";

// ---------------------------------------------------------------------------
// Date helpers
// ---------------------------------------------------------------------------

function parseDate(input: string): { apiDate: string; displayDate: string; shortDate: string } {
  const mmdd = input.match(/^(\d{1,2})\/(\d{1,2})$/);
  const mmddyyyy = input.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);

  let month: number, day: number, year: number;

  if (mmdd) {
    month = parseInt(mmdd[1], 10);
    day = parseInt(mmdd[2], 10);
    year = new Date().getFullYear();
  } else if (mmddyyyy) {
    month = parseInt(mmddyyyy[1], 10);
    day = parseInt(mmddyyyy[2], 10);
    year = parseInt(mmddyyyy[3], 10);
  } else {
    throw new Error(`Invalid date format: "${input}". Use MM/DD or MM/DD/YYYY.`);
  }

  // Validate
  const d = new Date(year, month - 1, day);
  if (d.getFullYear() !== year || d.getMonth() !== month - 1 || d.getDate() !== day) {
    throw new Error(`Invalid date: ${input}`);
  }

  const monthNames = [
    "January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December",
  ];

  return {
    apiDate: `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`,
    displayDate: `${monthNames[month - 1]} ${day}, ${year}`,
    shortDate: `${month}/${day}`,
  };
}

// ---------------------------------------------------------------------------
// Build event payload
// ---------------------------------------------------------------------------

function buildHappyHourPayload(apiDate: string, displayDate: string) {
  const datePart = process.env.EVENT_DATE ?? "";
  return {
    id: "",
    secret: "",
    title: `Westside Bike Happy Hour ${datePart}`,
    details:
      "Join us on the westside for Bike Happy Hour. Meet new friends and old, hang out, grab a beverage (alcoholic or not), grab some food, and let's talk bikes! \r\n\r\nEveryone welcome!\r\n\r\nEvery 2nd and 4th Monday, 4:30 to 7 p.m.",
    audience: "G",
    time: "16:30:00",
    timedetails: "4:30 to 7pm",
    eventduration: "",
    area: "W",
    venue: "BGs Food Cartel",
    address: "4250 SW Rose Biggi Ave Beaverton, OR",
    locdetails: "Meet in the back by the bar or in the indoor seating",
    locend: "",
    length: "--",
    organizer: "Ride Westside",
    email: "ridewestside2023@gmail.com",
    hideemail: "1",
    webname: "Ride Westside",
    weburl: "https://ridewestside.org",
    phone: "",
    contact: "",
    tinytitle: "",
    printdescr: "",
    code_of_conduct: "1",
    read_comic: "1",
    datestatuses: [{ id: "", date: apiDate, status: "A", newsflash: "" }],
  };
}

function buildCustomPayload(apiDate: string) {
  return {
    id: "",
    secret: "",
    title: process.env.EVENT_TITLE ?? "Ride Westside Event",
    details: process.env.EVENT_DETAILS ?? "",
    audience: process.env.EVENT_AUDIENCE ?? "G",
    time: process.env.EVENT_TIME ?? "10:00:00",
    timedetails: process.env.EVENT_TIME_DETAIL ?? "",
    eventduration: "",
    area: process.env.EVENT_AREA ?? "W",
    venue: process.env.EVENT_VENUE ?? "Beaverton Central MAX Station",
    address: process.env.EVENT_ADDRESS ?? "12700 SW Crescent St, Beaverton, OR 97005",
    locdetails: process.env.EVENT_LOC_DETAILS ?? "",
    locend: "",
    length: "--",
    organizer: "Ride Westside",
    email: "ridewestside2023@gmail.com",
    hideemail: "1",
    webname: "Ride Westside",
    weburl: "https://ridewestside.org",
    phone: "",
    contact: "",
    tinytitle: "",
    printdescr: "",
    code_of_conduct: "1",
    read_comic: "1",
    datestatuses: [{ id: "", date: apiDate, status: "A", newsflash: "" }],
  };
}

// ---------------------------------------------------------------------------
// Shift2Bikes API
// ---------------------------------------------------------------------------

interface Shift2BikesResponse {
  id?: string;
  datestatuses?: Array<{ id?: string }>;
}

async function submitEvent(payload: Record<string, unknown>): Promise<string> {
  console.log("Submitting event to Shift2Bikes...");
  console.log(`  Title: ${payload.title}`);
  console.log(`  Date:  ${(payload.datestatuses as Array<{ date: string }>)[0].date}`);

  const res = await fetch(SHIFT2BIKES_API, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Shift2Bikes API returned ${res.status}: ${body}`);
  }

  const data: Shift2BikesResponse = await res.json();
  console.log("API response:", JSON.stringify(data, null, 2));

  // Extract event ID from response
  const eventId =
    data.datestatuses?.[0]?.id ?? data.id;

  if (!eventId) {
    throw new Error("Could not extract event ID from API response.");
  }

  return String(eventId);
}

// ---------------------------------------------------------------------------
// Append to events.md
// ---------------------------------------------------------------------------

function appendEvent(opts: {
  title: string;
  displayDate: string;
  eventId: string;
  start?: string;
  end?: string;
  route?: string;
}) {
  const eventsPath = path.resolve("content/events.md");
  let content = fs.readFileSync(eventsPath, "utf-8");

  // Build the YAML block to insert before the closing ---
  const lines: string[] = [];
  lines.push("");
  lines.push(`  - title: "${opts.title}"`);
  lines.push(`    date: "${opts.displayDate}"`);
  lines.push(`    url: "https://shift2bikes.org/calendar/event-${opts.eventId}"`);
  if (opts.route) {
    lines.push(`    route: "${opts.route}"`);
  }
  if (opts.start) {
    lines.push(`    start: "${opts.start}"`);
  }
  if (opts.end) {
    lines.push(`    end: "${opts.end}"`);
  }

  const block = lines.join("\n");

  // Insert before the final ---
  const lastSeparator = content.lastIndexOf("\n---");
  if (lastSeparator === -1) {
    throw new Error("Could not find closing --- in events.md");
  }

  content = content.slice(0, lastSeparator) + block + content.slice(lastSeparator);

  fs.writeFileSync(eventsPath, content, "utf-8");
  console.log(`Appended event to ${eventsPath}`);

  // Return the URL for the workflow output
  return `https://shift2bikes.org/calendar/event-${opts.eventId}`;
}

// ---------------------------------------------------------------------------
// GitHub Actions output helper
// ---------------------------------------------------------------------------

function setOutput(name: string, value: string) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (outputFile) {
    fs.appendFileSync(outputFile, `${name}=${value}\n`);
  }
  console.log(`::set-output name=${name}::${value}`);
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main() {
  const eventType = process.env.EVENT_TYPE ?? "happy-hour";
  const dateInput = process.env.EVENT_DATE;

  if (!dateInput) {
    throw new Error("EVENT_DATE is required.");
  }

  const { apiDate, displayDate, shortDate } = parseDate(dateInput);

  let payload: Record<string, unknown>;
  let eventTitle: string;

  if (eventType === "happy-hour") {
    payload = buildHappyHourPayload(apiDate, displayDate);
    eventTitle = `${shortDate} Bike Happy Hour`;
  } else {
    payload = buildCustomPayload(apiDate);
    eventTitle = process.env.EVENT_TITLE ?? "Ride Westside Event";
  }

  const eventId = await submitEvent(payload);
  console.log(`Event created with ID: ${eventId}`);

  const eventUrl = appendEvent({
    title: eventTitle,
    displayDate,
    eventId,
    start: process.env.EVENT_START || (eventType === "happy-hour" ? "Beaverton" : undefined),
    end: process.env.EVENT_END || (eventType === "happy-hour" ? "Beaverton" : undefined),
    route: process.env.EVENT_ROUTE || undefined,
  });

  setOutput("event-id", eventId);
  setOutput("event-url", eventUrl);

  console.log("\n=== IMPORTANT ===");
  console.log("Check the Ride Westside Gmail for the confirmation email.");
  console.log("The event will NOT appear on the Shift2Bikes calendar until confirmed via email.");
  console.log(`Event URL: ${eventUrl}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
