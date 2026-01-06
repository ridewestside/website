/**
 * Ride Westside - Event filtering and collapsible sections
 */

interface EventElement {
  element: HTMLElement;
  date: Date;
}

interface FilterState {
  start: string;
  end: string;
}

const DAYS_THRESHOLD = 90;

function parseEventDate(dateStr: string): Date | null {
  // Parse dates like "January 12, 2026"
  const parsed = Date.parse(dateStr);
  if (isNaN(parsed)) {
    return null;
  }
  return new Date(parsed);
}

function initEventFiltering(): void {
  const eventsSection = document.querySelector('[data-section="events"]');
  const pastSection = document.querySelector('[data-section="past-events"]');
  const futureSection = document.querySelector('[data-section="future-events"]');

  if (!eventsSection) {
    return;
  }

  const eventCards = eventsSection.querySelectorAll<HTMLElement>('.event-card[data-date]');
  const now = new Date();
  now.setHours(0, 0, 0, 0);

  const futureThreshold = new Date(now);
  futureThreshold.setDate(futureThreshold.getDate() + DAYS_THRESHOLD);

  const pastEvents: EventElement[] = [];
  const upcomingEvents: EventElement[] = [];
  const futureEvents: EventElement[] = [];

  eventCards.forEach((card) => {
    const dateStr = card.getAttribute('data-date');
    if (!dateStr) return;

    const eventDate = parseEventDate(dateStr);
    if (!eventDate) return;

    if (eventDate < now) {
      pastEvents.push({ element: card, date: eventDate });
    } else if (eventDate > futureThreshold) {
      futureEvents.push({ element: card, date: eventDate });
    } else {
      upcomingEvents.push({ element: card, date: eventDate });
    }
  });

  // Move past events to past section
  if (pastSection) {
    const pastContainer = pastSection.querySelector('.collapsible-container');
    if (pastContainer && pastEvents.length > 0) {
      // Sort past events by date descending (most recent first)
      pastEvents.sort((a, b) => b.date.getTime() - a.date.getTime());

      pastEvents.forEach(({ element }) => {
        pastContainer.appendChild(element);
      });

      pastSection.classList.add('has-events');

      const countEl = pastSection.querySelector('.section-count');
      if (countEl) {
        countEl.textContent = `(${pastEvents.length})`;
      }
    }
  }

  // Move future events to future section
  if (futureSection) {
    const futureContainer = futureSection.querySelector('.collapsible-container');
    if (futureContainer && futureEvents.length > 0) {
      // Sort future events by date ascending (soonest first)
      futureEvents.sort((a, b) => a.date.getTime() - b.date.getTime());

      futureEvents.forEach(({ element }) => {
        futureContainer.appendChild(element);
      });

      futureSection.classList.add('has-events');

      const countEl = futureSection.querySelector('.section-count');
      if (countEl) {
        countEl.textContent = `(${futureEvents.length})`;
      }
    }
  }

  // If no upcoming events, show a message
  if (upcomingEvents.length === 0) {
    const noEventsMsg = document.createElement('p');
    noEventsMsg.className = 'no-events-message';
    noEventsMsg.textContent = 'No upcoming events scheduled. Check back soon!';
    eventsSection.appendChild(noEventsMsg);
  }
}

function initCollapsibleToggles(): void {
  const toggles = document.querySelectorAll<HTMLButtonElement>('.collapsible-toggle');

  toggles.forEach((toggle) => {
    const containerId = toggle.getAttribute('aria-controls');
    if (!containerId) return;

    const container = document.getElementById(containerId);
    if (!container) return;

    toggle.addEventListener('click', () => {
      const isExpanded = toggle.getAttribute('aria-expanded') === 'true';
      toggle.setAttribute('aria-expanded', String(!isExpanded));
      container.classList.toggle('expanded');

      const icon = toggle.querySelector('.toggle-icon');
      if (icon) {
        icon.textContent = isExpanded ? '+' : '-';
      }
    });
  });
}

function initLocationFilters(): void {
  const startSelect = document.getElementById('filter-start') as HTMLSelectElement | null;
  const endSelect = document.getElementById('filter-end') as HTMLSelectElement | null;
  const clearButton = document.getElementById('filter-clear') as HTMLButtonElement | null;

  if (!startSelect || !endSelect) {
    return;
  }

  const allEventCards = document.querySelectorAll<HTMLElement>('.event-card[data-start], .event-card[data-end]');

  // Collect unique start and end locations
  const startLocations = new Set<string>();
  const endLocations = new Set<string>();

  allEventCards.forEach((card) => {
    const start = card.getAttribute('data-start');
    const end = card.getAttribute('data-end');
    if (start) startLocations.add(start);
    if (end) endLocations.add(end);
  });

  // Populate dropdowns
  const sortedStarts = Array.from(startLocations).sort();
  const sortedEnds = Array.from(endLocations).sort();

  sortedStarts.forEach((loc) => {
    const option = document.createElement('option');
    option.value = loc;
    option.textContent = loc;
    startSelect.appendChild(option);
  });

  sortedEnds.forEach((loc) => {
    const option = document.createElement('option');
    option.value = loc;
    option.textContent = loc;
    endSelect.appendChild(option);
  });

  const filterState: FilterState = { start: '', end: '' };

  function applyFilters(): void {
    allEventCards.forEach((card) => {
      const cardStart = card.getAttribute('data-start') || '';
      const cardEnd = card.getAttribute('data-end') || '';

      const matchesStart = !filterState.start || cardStart === filterState.start;
      const matchesEnd = !filterState.end || cardEnd === filterState.end;

      if (matchesStart && matchesEnd) {
        card.classList.remove('filtered-out');
      } else {
        card.classList.add('filtered-out');
      }
    });

    // Show/hide clear button
    if (clearButton) {
      clearButton.style.display = (filterState.start || filterState.end) ? 'block' : 'none';
    }
  }

  startSelect.addEventListener('change', () => {
    filterState.start = startSelect.value;
    applyFilters();
  });

  endSelect.addEventListener('change', () => {
    filterState.end = endSelect.value;
    applyFilters();
  });

  if (clearButton) {
    clearButton.addEventListener('click', () => {
      filterState.start = '';
      filterState.end = '';
      startSelect.value = '';
      endSelect.value = '';
      applyFilters();
    });
  }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  initEventFiltering();
  initCollapsibleToggles();
  initLocationFilters();
});
