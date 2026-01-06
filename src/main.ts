/**
 * Ride Westside - Event filtering and collapsible sections
 */

import { parseISO, isBefore, isAfter, addDays, startOfDay } from "date-fns";

interface EventElement {
  element: HTMLElement;
  date: Date;
}

interface FilterState {
  start: string;
  end: string;
}

const DAYS_THRESHOLD = 90;
const FILTER_STORAGE_KEY = "ridewestside:filters";

/**
 * Parse event date string using date-fns
 */
function parseEventDate(dateStr: string): Date | null {
  try {
    // Try ISO format first, then fall back to natural language parsing
    const parsed = parseISO(dateStr);
    if (isNaN(parsed.getTime())) {
      // Fallback to native Date.parse for formats like "January 12, 2026"
      const fallback = Date.parse(dateStr);
      if (isNaN(fallback)) {
        return null;
      }
      return new Date(fallback);
    }
    return parsed;
  } catch {
    return null;
  }
}

/**
 * Load filter state from URL params or localStorage
 */
function loadFilterState(): FilterState {
  // Priority: URL params > localStorage > defaults
  const urlParams = new URLSearchParams(window.location.search);
  const urlStart = urlParams.get("start");
  const urlEnd = urlParams.get("end");

  // If URL has filters, use those
  if (urlStart !== null || urlEnd !== null) {
    return {
      start: urlStart || "",
      end: urlEnd || "",
    };
  }

  // Otherwise try localStorage
  try {
    const stored = localStorage.getItem(FILTER_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as FilterState;
      return parsed;
    }
  } catch (e) {
    console.warn("Failed to load filter state from localStorage:", e);
  }

  // Default empty filters
  return { start: "", end: "" };
}

/**
 * Save filter state to localStorage and update URL
 */
function saveFilterState(state: FilterState): void {
  // Save to localStorage
  try {
    localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(state));
  } catch (e) {
    console.warn("Failed to save filter state to localStorage:", e);
  }

  // Update URL without page reload
  const url = new URL(window.location.href);
  if (state.start) {
    url.searchParams.set("start", state.start);
  } else {
    url.searchParams.delete("start");
  }
  if (state.end) {
    url.searchParams.set("end", state.end);
  } else {
    url.searchParams.delete("end");
  }

  // Update URL without adding to history if nothing changed
  const newUrl = url.toString();
  if (newUrl !== window.location.href) {
    window.history.replaceState({}, "", newUrl);
  }
}

/**
 * Add location display to event cards
 */
function addLocationDisplay(card: HTMLElement): void {
  const startLoc = card.getAttribute("data-start");
  const endLoc = card.getAttribute("data-end");

  // Build location text
  let locationText = "";
  if (startLoc && endLoc) {
    locationText = `${startLoc} → ${endLoc}`;
  } else if (startLoc) {
    locationText = startLoc;
  } else if (endLoc) {
    locationText = endLoc;
  }

  // If we have location text, add it to the card
  if (locationText) {
    const linkButton = card.querySelector(".link-button");
    if (linkButton) {
      // Check if we already added a location span
      let locationSpan = linkButton.querySelector(".link-location");
      if (!locationSpan) {
        locationSpan = document.createElement("span");
        locationSpan.className = "link-location";
        linkButton.appendChild(locationSpan);
      }
      locationSpan.textContent = locationText;
    }
  }
}

function initEventFiltering(): void {
  const eventsSection = document.querySelector('[data-section="events"]');
  const pastSection = document.querySelector('[data-section="past-events"]');
  const futureSection = document.querySelector(
    '[data-section="future-events"]',
  );

  if (!eventsSection) {
    return;
  }

  const eventCards = eventsSection.querySelectorAll<HTMLElement>(
    ".event-card[data-date]",
  );
  const now = startOfDay(new Date());
  const futureThreshold = addDays(now, DAYS_THRESHOLD);

  const pastEvents: EventElement[] = [];
  const upcomingEvents: EventElement[] = [];
  const futureEvents: EventElement[] = [];

  eventCards.forEach((card) => {
    const dateStr = card.getAttribute("data-date");
    if (!dateStr) return;

    const eventDate = parseEventDate(dateStr);
    if (!eventDate) return;

    if (isBefore(eventDate, now)) {
      pastEvents.push({ element: card, date: eventDate });
    } else if (isAfter(eventDate, futureThreshold)) {
      futureEvents.push({ element: card, date: eventDate });
    } else {
      upcomingEvents.push({ element: card, date: eventDate });
    }
  });

  // Sort upcoming events by date ascending (soonest first)
  upcomingEvents.sort((a, b) => a.date.getTime() - b.date.getTime());

  // Add location displays and re-append upcoming events in sorted order
  upcomingEvents.forEach(({ element }) => {
    addLocationDisplay(element);
    eventsSection.appendChild(element);
  });

  // Move past events to past section
  if (pastSection) {
    const pastContainer = pastSection.querySelector(".collapsible-container");
    if (pastContainer && pastEvents.length > 0) {
      // Sort past events by date descending (most recent first)
      pastEvents.sort((a, b) => b.date.getTime() - a.date.getTime());

      pastEvents.forEach(({ element }) => {
        addLocationDisplay(element);
        pastContainer.appendChild(element);
      });

      pastSection.classList.add("has-events");

      const countEl = pastSection.querySelector(".section-count");
      if (countEl) {
        countEl.textContent = `(${pastEvents.length})`;
      }
    }
  }

  // Move future events to future section
  if (futureSection) {
    const futureContainer = futureSection.querySelector(
      ".collapsible-container",
    );
    if (futureContainer && futureEvents.length > 0) {
      // Sort future events by date ascending (soonest first)
      futureEvents.sort((a, b) => a.date.getTime() - b.date.getTime());

      futureEvents.forEach(({ element }) => {
        addLocationDisplay(element);
        futureContainer.appendChild(element);
      });

      futureSection.classList.add("has-events");

      const countEl = futureSection.querySelector(".section-count");
      if (countEl) {
        countEl.textContent = `(${futureEvents.length})`;
      }
    }
  }

  // If no upcoming events, show a message
  if (upcomingEvents.length === 0) {
    const noEventsMsg = document.createElement("p");
    noEventsMsg.className = "no-events-message";
    noEventsMsg.textContent = "No upcoming events scheduled. Check back soon!";
    eventsSection.appendChild(noEventsMsg);
  }
}

function initCollapsibleToggles(): void {
  const toggles = document.querySelectorAll<HTMLButtonElement>(
    ".collapsible-toggle",
  );

  toggles.forEach((toggle) => {
    const containerId = toggle.getAttribute("aria-controls");
    if (!containerId) return;

    const container = document.getElementById(containerId);
    if (!container) return;

    toggle.addEventListener("click", () => {
      const isExpanded = toggle.getAttribute("aria-expanded") === "true";
      toggle.setAttribute("aria-expanded", String(!isExpanded));
      container.classList.toggle("expanded");

      const icon = toggle.querySelector(".toggle-icon");
      if (icon) {
        icon.textContent = isExpanded ? "+" : "-";
      }
    });
  });
}

function initLocationFilters(): void {
  const startSelect = document.getElementById(
    "filter-start",
  ) as HTMLSelectElement | null;
  const endSelect = document.getElementById(
    "filter-end",
  ) as HTMLSelectElement | null;
  const clearButton = document.getElementById(
    "filter-clear",
  ) as HTMLButtonElement | null;

  if (!startSelect || !endSelect) {
    return;
  }

  const allEventCards = document.querySelectorAll<HTMLElement>(
    ".event-card[data-start], .event-card[data-end]",
  );

  // Collect unique start and end locations
  const startLocations = new Set<string>();
  const endLocations = new Set<string>();

  allEventCards.forEach((card) => {
    const start = card.getAttribute("data-start");
    const end = card.getAttribute("data-end");
    if (start) startLocations.add(start);
    if (end) endLocations.add(end);
  });

  // Populate dropdowns
  const sortedStarts = Array.from(startLocations).sort();
  const sortedEnds = Array.from(endLocations).sort();

  sortedStarts.forEach((loc) => {
    const option = document.createElement("option");
    option.value = loc;
    option.textContent = loc;
    startSelect.appendChild(option);
  });

  sortedEnds.forEach((loc) => {
    const option = document.createElement("option");
    option.value = loc;
    option.textContent = loc;
    endSelect.appendChild(option);
  });

  // Load initial filter state from URL or localStorage
  const filterState: FilterState = loadFilterState();

  // Apply initial values to dropdowns
  if (filterState.start && sortedStarts.includes(filterState.start)) {
    startSelect.value = filterState.start;
  }
  if (filterState.end && sortedEnds.includes(filterState.end)) {
    endSelect.value = filterState.end;
  }

  function applyFilters(): void {
    allEventCards.forEach((card) => {
      const cardStart = card.getAttribute("data-start") || "";
      const cardEnd = card.getAttribute("data-end") || "";

      const matchesStart =
        !filterState.start || cardStart === filterState.start;
      const matchesEnd = !filterState.end || cardEnd === filterState.end;

      if (matchesStart && matchesEnd) {
        card.classList.remove("filtered-out");
      } else {
        card.classList.add("filtered-out");
      }
    });

    // Show/hide clear button
    if (clearButton) {
      clearButton.style.display =
        filterState.start || filterState.end ? "block" : "none";
    }

    // Save state to localStorage and URL
    saveFilterState(filterState);
  }

  // Apply filters on page load if state was restored
  if (filterState.start || filterState.end) {
    applyFilters();
  }

  startSelect.addEventListener("change", () => {
    filterState.start = startSelect.value;
    applyFilters();
  });

  endSelect.addEventListener("change", () => {
    filterState.end = endSelect.value;
    applyFilters();
  });

  if (clearButton) {
    clearButton.addEventListener("click", () => {
      filterState.start = "";
      filterState.end = "";
      startSelect.value = "";
      endSelect.value = "";
      applyFilters();
    });
  }
}

// Initialize when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
  initEventFiltering();
  initCollapsibleToggles();
  initLocationFilters();
});
