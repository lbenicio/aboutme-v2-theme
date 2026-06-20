# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-06-20

### Added

- **SCSS framework** — replaced Tailwind CSS and PostCSS with a custom SCSS framework processed natively by Hugo Pipes (`css.Sass`). No Node.js required.
- **Design tokens** — CSS custom properties for colors, spacing, typography, shadows, and transitions in `_variables.scss`.
- **Utility-first SCSS** — 14 partials covering reset, typography, layout, spacing, sizing, colors, borders, effects, interactive, components, dark mode, responsive, and dynamic styles.
- **Go CLI tools** in `cmd/`:
  - `obfuscate` — deterministic class/id/data-attribute obfuscator for built HTML/CSS/JS.
  - `ogimage` — generates 1200×630 OG social preview cards (also available as Hugo-native partial).
  - `release` — changelog generator from conventional commits.
  - `analytics` — event sender to HTTP endpoint.
- **Protected names** in obfuscator for critical classes (`sr-only`, `skip-to-content`), data attributes (contact form, PGP, analytics), and element IDs (theme toggle, mobile menu).
- **Global button reset** in SCSS — `appearance: none`, no border/background/padding on all `<button>` elements.

### Changed

- **About page**: converted from list (`_index.md`) to leaf bundle single page (`index.md`). Template moved to `layouts/about/single.html`.
- **Timeline page**: converted from list (`_index.md`) to leaf bundle single page (`index.md`). Template moved to `layouts/timeline/single.html`.
- **Contact page**: already a single page; template at `layouts/contact/single.html`.
- **Directory structure**: static files under `static/static/` (served with `/static/` URL prefix), processed assets under `assets/assets/` (served with `/assets/` URL prefix).
- **Header styling**: theme toggle button simplified to match nav link appearance; navbar uses `gap` utilities for spacing.
- **Font loading**: fonts served from `assets/assets/fonts/` via Hugo Pipes; preload uses `resources.Get`.
- **CSS delivery**: no fingerprint or integrity hashes (to avoid GitHub Pages CDN mismatch); served at `/assets/scss/main.min.css` and `/assets/css/fonts.min.css`.
- **Theme config**: reduced `theme.toml` to essentials — only social handler defaults remain; all page content sourced from front matter.
- **Hugo version**: minimum `0.163.0` (uses `css.Sass` instead of deprecated `resources.ToCSS`).

### Removed

- **Node.js dependency**: deleted `package.json`, `package-lock.json`, `node_modules/`, `tailwind.config.cjs`, `postcss.config.cjs`, `tsconfig.json`, `eslint.config.ts`, `vitest.config.ts`, `playwright.config.ts`, all `scripts/*.mjs`, and `tests/`.

### Fixed

- Obfuscator: prevent class name substring replacement inside `querySelector` attribute selectors (e.g. `[data-pgp-block]` broken by `block` class renaming).
- Obfuscator: handle Hugo-minified unquoted HTML attributes (`class=sr-only`).
- Obfuscator: sort CSS replacements by length to prevent partial prefix matches.
- CSS: use `\Q...\E` literal matching for selectors with special characters.
- JS: use single-quoted strings in inline scripts to avoid `jsonify` double-quote nesting.
- CDN: removed fingerprinting to prevent SRI mismatch on GitHub Pages.
- SCSS: use uppercase `HSL()` to avoid LibSass crash on `hsl(var(--))`.
