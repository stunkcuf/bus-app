# Frontend Build Process

This document describes the frontend build system for the Fleet Management System.

## Overview

The build system provides:
- **Asset bundling and minification** for production
- **Development builds** with source maps
- **Hot reloading** during development
- **Cache busting** with content hashes
- **Performance optimizations** like tree shaking

## Prerequisites

- Node.js 16+ and npm
- Go 1.19+

## Quick Start

1. **Install dependencies:**
   ```bash
   make frontend-install
   # or
   npm install
   ```

2. **Development mode:**
   ```bash
   make dev
   # This starts both frontend watching and the Go server
   ```

3. **Production build:**
   ```bash
   make build-prod
   # This builds optimized frontend assets and Go binary
   ```

## Build Commands

### Frontend Only

- `make frontend-install` - Install Node.js dependencies
- `make frontend` - Build production assets
- `make frontend-dev` - Build development assets
- `make frontend-watch` - Watch for changes and rebuild
- `make frontend-clean` - Clean build artifacts

### Full Application

- `make build-prod` - Complete production build
- `make build-dev` - Complete development build
- `make dev` - Development mode with watching
- `make dev-air` - Development with Air hot reloading (if available)

### NPM Scripts

You can also use npm directly:

- `npm run build` - Production build
- `npm run build:dev` - Development build
- `npm run watch` - Watch mode
- `npm run clean` - Clean build artifacts

## Build Configuration

### Webpack Configuration

The build system uses Webpack 5 with:

- **Entry Points:**
  - `app` - Main application bundle
  - `vendor` - Third-party libraries
  - `wizards` - Wizard-specific functionality
  - `styles` - Main CSS bundle

- **Output:**
  - Development: `dist/[name].js`, `dist/[name].css`
  - Production: `dist/[name].[contenthash].js`, `dist/[name].[contenthash].css`

- **Optimizations:**
  - Code splitting
  - Tree shaking
  - Minification
  - Source maps
  - Cache busting

### Build Targets

- **Modern browsers:** > 1%, last 2 versions
- **ES2017+** for modern JavaScript features
- **CSS Grid** and **Flexbox** support

## File Structure

```
├── src/
│   ├── js/
│   │   └── app.js          # Main entry point
│   └── css/
│       └── main.css        # Main stylesheet
├── static/                 # Source assets
│   ├── *.js               # JavaScript modules
│   └── *.css              # Stylesheets
├── dist/                   # Built assets (generated)
│   ├── app.[hash].js      # Main bundle
│   ├── vendor.[hash].js   # Vendor bundle
│   ├── wizards.[hash].js  # Wizard bundle
│   └── styles.[hash].css  # Stylesheet bundle
└── node_modules/          # Dependencies
```

## Development Workflow

1. **Start development:**
   ```bash
   make dev
   ```

2. **Make changes to files in `static/`**
   - JavaScript files are watched and bundled automatically
   - CSS files are processed and bundled
   - Go files trigger server restart (with Air)

3. **View changes:**
   - Frontend changes rebuild automatically
   - Browser refresh may be needed
   - Check console for build errors

## Production Deployment

1. **Build production assets:**
   ```bash
   make build-prod
   ```

2. **Deploy files:**
   - Binary: `hs-bus` (or `hs-bus.exe` on Windows)
   - Assets: `dist/` directory
   - Templates: `templates/` directory

3. **Server configuration:**
   - Serve static files from `/dist/` route
   - Enable gzip compression
   - Set cache headers for hashed assets

## Performance Features

### Code Splitting
- Vendor libraries are bundled separately
- Wizard code is loaded only when needed
- Common code is extracted to reduce duplication

### Asset Optimization
- JavaScript minification and mangling
- CSS minification and optimization
- Image compression (future enhancement)
- Font subsetting (future enhancement)

### Caching Strategy
- Content-based hashing for cache busting
- Long-term caching for static assets
- Immutable assets with proper headers

## Browser Support

- **Modern browsers:** Chrome 88+, Firefox 85+, Safari 14+, Edge 88+
- **Fallbacks:** Basic functionality on older browsers
- **Progressive enhancement:** Enhanced features for modern browsers

## Troubleshooting

### Build Errors

1. **"Module not found" errors:**
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   ```

2. **Permission errors:**
   ```bash
   sudo chown -R $(whoami) node_modules/
   ```

3. **Out of memory errors:**
   ```bash
   export NODE_OPTIONS="--max-old-space-size=4096"
   npm run build
   ```

### Performance Issues

1. **Slow builds:**
   - Use `make frontend-dev` for development
   - Enable webpack cache: `cache: true` in config

2. **Large bundle sizes:**
   - Run `npm run analyze` to inspect bundles
   - Consider code splitting for large dependencies

### Development Issues

1. **Changes not reflecting:**
   - Check webpack watch is running
   - Clear browser cache
   - Restart development server

2. **Hot reloading not working:**
   - Ensure file watching is enabled
   - Check file permissions
   - Use polling if on network drive

## Environment Variables

- `NODE_ENV=production` - Production build mode
- `NODE_ENV=development` - Development build mode
- `WEBPACK_MODE=production` - Override webpack mode

## Future Enhancements

- [ ] Hot module replacement (HMR)
- [ ] Service worker for offline support
- [ ] Image optimization pipeline
- [ ] CSS preprocessor support (SASS/LESS)
- [ ] Bundle analyzer integration
- [ ] Automated testing integration
- [ ] TypeScript support
- [ ] ESLint and Prettier integration

## Integration with Go Server

The Go server serves built assets from the `/dist/` directory:

```go
// Static files - serve from dist in production, static in development
if os.Getenv("APP_ENV") == "production" {
    fs := http.FileServer(http.Dir("./dist/"))
    mux.Handle("/dist/", http.StripPrefix("/dist/", fs))
} else {
    fs := http.FileServer(http.Dir("./static/"))
    mux.Handle("/static/", http.StripPrefix("/static/", fs))
}
```

In templates, use:
```html
<!-- Development -->
<link rel="stylesheet" href="/static/main.css">
<script src="/static/app.js"></script>

<!-- Production -->
<link rel="stylesheet" href="/dist/styles.[hash].css">
<script src="/dist/app.[hash].js"></script>
```