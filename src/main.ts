/**
 * Ride Westside - Event filtering and Past Rides toggle
 */

interface EventElement {
  element: HTMLAnchorElement;
  date: Date;
}

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

  if (!eventsSection || !pastSection) {
    return;
  }

  const eventLinks = eventsSection.querySelectorAll<HTMLAnchorElement>('.link-button[data-date]');
  const now = new Date();
  now.setHours(0, 0, 0, 0); // Compare dates only, not times

  const pastEvents: EventElement[] = [];
  const upcomingEvents: EventElement[] = [];

  eventLinks.forEach((link) => {
    const dateStr = link.getAttribute('data-date');
    if (!dateStr) return;

    const eventDate = parseEventDate(dateStr);
    if (!eventDate) return;

    if (eventDate < now) {
      pastEvents.push({ element: link, date: eventDate });
    } else {
      upcomingEvents.push({ element: link, date: eventDate });
    }
  });

  // Move past events to past section
  const pastContainer = pastSection.querySelector('.past-events-container');
  if (pastContainer && pastEvents.length > 0) {
    // Sort past events by date descending (most recent first)
    pastEvents.sort((a, b) => b.date.getTime() - a.date.getTime());

    pastEvents.forEach(({ element }) => {
      pastContainer.appendChild(element);
    });

    // Show the past events section
    pastSection.classList.add('has-events');

    // Update the count
    const countEl = pastSection.querySelector('.past-events-count');
    if (countEl) {
      countEl.textContent = `(${pastEvents.length})`;
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

function initPastEventsToggle(): void {
  const toggle = document.querySelector<HTMLButtonElement>('.past-events-toggle');
  const container = document.querySelector('.past-events-container');

  if (!toggle || !container) {
    return;
  }

  toggle.addEventListener('click', () => {
    const isExpanded = toggle.getAttribute('aria-expanded') === 'true';
    toggle.setAttribute('aria-expanded', String(!isExpanded));
    container.classList.toggle('expanded');

    const icon = toggle.querySelector('.toggle-icon');
    if (icon) {
      icon.textContent = isExpanded ? '+' : '-';
    }
  });
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  initEventFiltering();
  initPastEventsToggle();
});
