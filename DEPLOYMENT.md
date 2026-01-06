# Deployment Guide

This document explains how to build and deploy the Ride Westside website, including the new TypeScript dependencies and build process.

## Prerequisites

- **Hugo** (v0.140.1 or later, extended version)
- **Go** (for Mage build tool)
- **Node.js** (v20 or later)
- **npm** (comes with Node.js)
- **esbuild** (installed globally or via npm)
- **Mage** (Go task runner)

## Local Development Setup

### 1. Install Dependencies

```bash
# Install npm dependencies (includes date-fns)
npm install

# Install esbuild globally (if not already installed)
npm install -g esbuild
```

### 2. Build the Site

Using Mage (recommended):

```bash
# Build everything (TypeScript + Hugo)
mage build

# Or run individual tasks
mage buildts  # Build TypeScript only
mage serve    # Build and start Hugo dev server
mage dev      # Build TypeScript and start Hugo dev server
mage watch    # Watch TypeScript files for changes
```

Manual build:

```bash
# Build TypeScript
esbuild src/main.ts \
  --bundle \
  --minify \
  --sourcemap \
  --target=es2020 \
  --outfile=themes/linkpage/static/js/main.js

# Build Hugo site
hugo --gc --minify
```

### 3. Development Workflow

**Option 1: Single Terminal (Recommended for most cases)**
```bash
mage serve
# Make TypeScript changes, then rebuild with:
mage buildts
```

**Option 2: Two Terminals (Auto-rebuild TypeScript)**
```bash
# Terminal 1: Watch TypeScript
mage watch

# Terminal 2: Run Hugo server
mage serve
```

## Build Process

### Magefile Targets

The `magefiles/magefile.go` defines the following build targets:

- **`InstallNpmDeps`**: Installs npm dependencies (date-fns, etc.)
- **`BuildTS`**: Compiles TypeScript to JavaScript (depends on InstallNpmDeps)
- **`Build`**: Full build - TypeScript + Hugo site (default target)
- **`Serve`**: Build TypeScript and start Hugo dev server
- **`Dev`**: Alias for Serve with helpful message
- **`Watch`**: Watch TypeScript files and rebuild on changes
- **`CheckLinks`**: Build site and check for dead external links
- **`Clean`**: Remove the `public/` directory

### Dependency Chain

```
Build → BuildTS → InstallNpmDeps
         ↓
    Compile TypeScript with esbuild
         ↓
    themes/linkpage/static/js/main.js
```

## GitHub Actions Deployment

The site automatically deploys to GitHub Pages when changes are pushed to the `main` branch.

### Workflow Steps (`.github/workflows/hugo.yml`)

1. **Install Hugo CLI** - Downloads and installs Hugo extended
2. **Checkout** - Clones the repository with submodules
3. **Setup Node.js** - Installs Node.js 20 with npm caching
4. **Install dependencies** - Runs `npm install` to get date-fns
5. **Install esbuild** - Installs esbuild globally
6. **Build TypeScript** - Bundles and minifies TypeScript → JavaScript
7. **Setup Pages** - Configures GitHub Pages
8. **Build with Hugo** - Generates static site
9. **Upload artifact** - Uploads `public/` directory
10. **Deploy to GitHub Pages** - Publishes the site

### Key Features

- **npm caching**: Dependencies are cached between runs for faster builds
- **Automatic builds**: Triggered on push to `main` or manual dispatch
- **TypeScript compilation**: Runs before Hugo to ensure JS is available
- **date-fns bundling**: The library is bundled into the output JavaScript

## TypeScript Features

### Dependencies

The `package.json` includes:

- **date-fns** (v3.3.1): Modern date manipulation library
  - Used for parsing dates, date comparisons, and date arithmetic
  - Tree-shakeable (only imports used functions)
  - Strong TypeScript support

### Features in `src/main.ts`

1. **Event Filtering**: Categorizes events into past/upcoming/future
2. **Location Filters**: Filter events by start/end location
3. **Collapsible Sections**: Toggle visibility of event groups
4. **URL State Sync**: Filter selections persist in URL query params
5. **localStorage**: Remembers user's last filter settings
6. **date-fns Integration**: Clean date parsing and comparison

### State Persistence

The site now remembers filter settings via:

1. **URL Parameters**: `?start=Venice&end=Malibu`
   - Shareable links with filters pre-applied
   - Priority over localStorage
   
2. **localStorage**: `ridewestside:filters`
   - Persists between visits
   - Used when no URL params present

Priority: URL params → localStorage → empty defaults

## Troubleshooting

### npm install fails

```bash
# Clear npm cache
npm cache clean --force

# Remove node_modules and package-lock.json
rm -rf node_modules package-lock.json

# Reinstall
npm install
```

### TypeScript build fails

```bash
# Check if esbuild is installed
which esbuild

# Install globally if missing
npm install -g esbuild

# Verify TypeScript syntax
npm run type-check
```

### GitHub Actions build fails

Check the Actions tab on GitHub for detailed logs. Common issues:

- **Missing node_modules**: Ensure `npm install` step runs before TypeScript build
- **esbuild not found**: Ensure esbuild installation step runs
- **Import errors**: Verify date-fns is in package.json dependencies

### Date filtering not working

1. Check browser console for errors
2. Verify `data-date` attributes on event cards
3. Ensure JavaScript is loaded: `themes/linkpage/static/js/main.js`
4. Check that date-fns is bundled in the output JS

## Production Build

For a production-ready build:

```bash
# Set environment variables
export HUGO_ENVIRONMENT=production
export TZ=America/Los_Angeles

# Build
mage build

# Or manually
mage buildts
hugo --gc --minify

# Output is in public/
```

## Clean Build

To start fresh:

```bash
mage clean
rm -rf node_modules
npm install
mage build
```

## Questions?

For issues or questions, check:

- GitHub Actions logs (for deployment issues)
- Browser console (for JavaScript errors)
- Hugo output (for build warnings)
- `mage -l` (list available targets)