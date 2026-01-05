# Ride Westside Website

A Hugo-powered link page for Ride Westside, automatically deployed to GitHub Pages at [beta.ridewestside.org](https://beta.ridewestside.org).

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

This project uses [mise](https://mise.jdx.dev/) to manage Go and tool dependencies.

```bash
mise install          # Install Go and mage
mise exec -- mage -l  # List available tasks
```

### Available Tasks

| Task | Description |
|------|-------------|
| `mage build` | Build the Hugo site |
| `mage serve` | Start the Hugo development server |
| `mage checkLinks` | Check for dead links in the site |
| `mage clean` | Remove the public directory |

Or use Hugo directly:

```bash
hugo server -D
```

Visit http://localhost:1313 to preview changes.

## Deployment

The site automatically deploys to GitHub Pages when changes are pushed to the `main` branch.

### DNS Configuration

For the custom domain to work, configure your DNS with:

- **CNAME record**: `beta` pointing to `ridewestside.github.io`

Or for apex domain, add A records pointing to GitHub's IPs:
- 185.199.108.153
- 185.199.109.153
- 185.199.110.153
- 185.199.111.153
