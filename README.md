# Ride Westside Website

A Hugo-powered link page for Ride Westside, automatically deployed to GitHub Pages at [ridewestside.org](https://ridewestside.org).

## Managing Content

### Updating Events

Edit `content/events.md` to add, remove, or modify upcoming events:

```yaml
events:
  - title: "Event Name"
    date: "January 15, 2026"
    url: "https://example.com/event-link"
```

### Updating Articles/Press Coverage

Edit `content/articles.md` to manage press coverage:

```yaml
articles:
  - source: "Publication Name - Date"
    url: "https://example.com/article"
    quote: "Optional notable quote from the article"
```

### Social Links

Social media links are configured in `hugo.toml`:

```toml
[params.social]
  instagram = "https://instagram.com/ride_westside"
  facebook = "https://www.facebook.com/..."
  bluesky = "https://bsky.app/profile/..."
```

### Google Analytics

To enable click tracking:

1. Create a Google Analytics 4 property
2. Replace `G-XXXXXXXXXX` in `hugo.toml` with your tracking ID:

```toml
[services.googleAnalytics]
  ID = 'G-YOUR-TRACKING-ID'
```

All link clicks are automatically tracked with event labels.

### Adding a Logo

Place your logo image at `static/images/logo.png` (recommended size: 192x192px).

## Local Development

This project uses [mise](https://mise.jdx.dev/) to manage Go, Node.js, and tool dependencies.

```bash
mise install          # Install Go, Node.js, mage, esbuild, typescript
mise exec -- mage -l  # List available tasks
```

### Available Tasks

| Task | Description |
|------|-------------|
| `mage build` | Build TypeScript and Hugo site |
| `mage buildts` | Compile TypeScript only |
| `mage serve` | Start Hugo dev server (builds TS first) |
| `mage dev` | Development mode with Hugo server |
| `mage watch` | Watch TypeScript files for changes |
| `mage checkLinks` | Check for dead links in the site |
| `mage clean` | Remove the public directory |

For development, run in two terminals:
```bash
mise exec -- mage watch  # Terminal 1: watch TypeScript
mise exec -- mage serve  # Terminal 2: Hugo server
```

Visit http://localhost:1313 to preview changes.

## Deployment

The site automatically deploys to GitHub Pages when changes are pushed to the `main` branch.

### DNS Configuration

Cloudflare: A, AAAA, and CNAME records as required by GitHub docs.
