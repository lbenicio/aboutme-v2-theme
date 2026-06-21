# AboutMe v2 Theme

A modern, production-ready Hugo theme for personal websites, portfolios, and blogs. Built with Tailwind CSS v4, TypeScript, and optimized for performance with multi-variant export support.

## ✨ Features

- **Modern Design**: Clean, responsive layout with dark mode support
- **Tailwind CSS v4**: Latest CSS-first configuration with lightning-fast builds via Lightning CSS
- **TypeScript**: Type-safe templates and components for maintainability
- **Multi-Variant Export**: Generate multiple site variants (blog, about-only, etc.) from a single codebase
- **Built-in Search**: Fast, client-side search for content discovery
- **SEO Optimized**: Comprehensive meta tags, Open Graph, Twitter cards, and structured data
- **Performance**: Optimized assets, lazy loading, and efficient caching strategies
- **Accessibility**: WCAG 2.1 compliant with semantic HTML and ARIA labels
- **Contact Form**: Encrypted PGP contact form with spam protection
- **Analytics Integration**: Built-in support for Umami and custom analytics
- **Content Sections**: About, Timeline, Publications, Reading List, Blog Posts, and Contact
- **Responsive Images**: AVIF/WebP/PNG fallbacks with automatic optimization
- **Internationalization**: Multi-language support with i18n ready structure

## 🚀 Quick Start

### Prerequisites

- **Hugo Extended** v0.152.0 or higher
- **Node.js** v25.0.0 or higher
- **npm** (comes with Node.js)

### Installation

1. **Create a new Hugo site**:

    ```bash
    hugo new-site my-site
    ```

    ```bash
    cd my-site
    ```

2. **Add the theme**:
   Option 1: Git submodule (recommended for development)

    ```bash
    git submodule add https://github.com/lbenicio/aboutme-v2-theme.git themes/aboutme-v2-theme
    ```

    Option 2: Clone directly

    ```bash
    git clone https://github.com/lbenicio/aboutme-v2-theme.git themes/aboutme-v2-theme
    ```

3. **Configure your site**:
   Add to your `hugo.toml`:

    ```toml
    theme = "aboutme-v2-theme"
    ```

4. **Install dependencies**:

    ```bash
    cd themes/aboutme-v2-theme
    ```

    ```bash
    npm install
    ```

5. **Start the development server**:

    From the theme directory

    ```bash
    npm run start:dev
    ```

    Or from your site root

    ```bash
    hugo server --themesDir ../..
    ```

## 📁 Project Structure

```
aboutme-v2-theme/
├── assets/
│   ├── css/  Tailwind CSS entry point
│   └── js/   JavaScript components
├── layouts/
│   ├── _default/ Default Hugo templates
│   ├── about/   About page section
│   ├── contact/ Contact page section
│   ├── partials/Reusable components
│   ├── post/ Blog post templates
│   ├── publications/  # Publications section
│   ├── reading/ Reading list section
│   ├── shortcodes/    # Hugo shortcodes
│   └── timeline/Timeline section
├── static/
│   ├── assets/  Static assets (images, fonts)
│   └── fonts/   Custom fonts
├── scripts/  Build and utility scripts
├── tests/    Unit and E2E tests
├── theme.toml   Theme configuration
└── package.json Node.js dependencies
```

## ⚙️ Configuration

### Theme Parameters

Add these to your site's `hugo.toml` under `[params]`:

```toml
[params]
  # Homepage
  homepageHeading = "About Your Name"
  homepageSubtitle = "Your personal tagline or description"

  # Profile Picture
  profilePicture = "static/assets/images/avatar/profile-picture.png"
  profilePictureAvif = "static/assets/images/avatar/profile-picture.avif"
  profilePictureWebp = "static/assets/images/avatar/profile-picture.webp"
  profilePictureAlt = "Your Name"

  # Site Mode
  appMode = "FULL"  # Options: "FULL", "BLOG_ONLY", "ABOUT_ONLY"
  siteUrl = "https://yourdomain.com"
  aboutOrigin = "https://yourdomain.com"

  # Contact Form
  contactEmail = "your@email.com"
  contactName = "Your Name"
  contactPostUrl = "https://formbold.com/s/your-form-id"
  contactAuthToken = "your-form-token"

  # Social Media
  githubHandler = "yourusername"
  linkedinHandler = "your-profile"
  twitterHandler = "yourusername"

  # Analytics
  umamiWebsiteId = "your-website-id"
  umamiScriptUrl = "https://your-analytics-url/script.js"

  # Newsletter
  newsletterPostUrl = "https://your-newsletter-url"
  newsletterCheckboxId = "your-checkbox-id"
  newsletterCheckboxValue = "your-checkbox-value"
  newsletterCheckboxName = "newsletter"

  # PGP (optional)
  pgpPublicKey = "your-pgp-public-key"

  # Fork Me Ribbon
  enableForkMe = true
  forkMeRepoUrl = "https://github.com/yourusername/your-repo"

  # Skills and Interests (displayed on About page)
  skills = ["Skill 1", "Skill 2", "Skill 3"]
  interests = ["Interest 1", "Interest 2", "Interest 3"]
  researchInterests = ["Research Interest 1", "Research Interest 2"]
```

### Content Configuration

The theme uses frontmatter in your content files to configure data structures:

#### Timeline (`content/timeline/_index.md`)

```yaml
---
Title: "Timeline"
description: "A chronological timeline of milestones"
timeline:
    - date: 2024-06
      title: "Your Milestone"
      category: career
      description: "Description of the milestone"
      links:
          - label: "Related Link"
            href: "/related-page/"
      highlight: true
---
```

#### Certifications (`content/about/_index.md`)

```yaml
---
Title: "About Me"
description: "About page description"
certifications:
    - id: aws-cloud-practitioner
      name: AWS Certified Cloud Practitioner
      issuer: Amazon Web Services
      level: Foundational
      issued: 2024-06
      domainTags:
          - cloud
          - aws
      skills:
          - Cloud Concepts
          - Security
      url: https://aws.amazon.com/certification/
      notes: "Additional notes"
---
```

## 🎨 Customization

### Styling

The theme uses Tailwind CSS v4 with a CSS-first configuration. To customize:

1. **Edit the main CSS file**: `assets/css/main.css`
2. **Modify theme tokens**: Update CSS custom properties in the `@layer base` section
3. **Add custom utilities**: Extend the `@theme` section with your design tokens

### Fonts

The theme includes Open Sans by default. To use custom fonts:

1. Place your font files in `static/fonts/`
2. Update the font paths in your `hugo.toml`:
    ```toml
    [params]
    fontRegular = "static/fonts/YourFont-Regular.ttf"
    fontBold = "static/fonts/YourFont-Bold.ttf"
    ```

### Images

Place your images in the appropriate directories:

- **Profile pictures**: `static/assets/images/avatar/`
- **OG images**: `static/assets/og/`
- **Favicons**: `static/assets/favicon/`

Supported formats: AVIF, WebP, PNG (with automatic fallbacks)

## 📦 Available Scripts

The theme includes several npm scripts for development and building:

Start Hugo dev server with hot reload

```bash
npm run start:dev
```

Start preview server

```bash
npm run start:preview
```

Build with Hugo

```bash
npm run build:hugo
```

Obfuscate inline scripts

```bash
npm run build:obfuscate
```

Clean build artifacts

```bash
npm run build:clean
```

Full production build

```bash
npm run build:prod
```

Run unit tests

```bash
npm run test:unit
```

Run UI tests

```bash
npm run test:ui
```

Run E2E tests with Playwright

```bash
npm run test:e2e
```

Run ESLint

```bash
npm run fmt:lint
```

Run Prettier

```bash
npm run fmt:format
```

Run both linting and formatting

```bash
npm run fmt:all
```

TypeScript type checking

```bash
npm run type:check
```

Clean build directories

```bash
npm run clear:build
```

Generate OG images

```bash
npm run og:gen
```

Generate PDF exports

```bash
npm run pdf:light
```

Generate book PDFs

```bash
npm run pdf:books
```

Generate release

```bash
npm run release:gen
```

## 🧪 Testing

The theme includes comprehensive tests:

- **Unit Tests**: Vitest for JavaScript/TypeScript utilities
- **UI Tests**: Vitest for template rendering
- **E2E Tests**: Playwright for full user flows

Run all tests:

```bash
npm run test:unit && npm run test:ui && npm run test:e2e
```

## 🌐 Multi-Variant Export

The theme supports generating multiple site variants from a single codebase:

Generate specific variant

```bash
node scripts/export.mjs --variant blog
```

```bash
node scripts/export.mjs --variant about-only
```

Available variants:

- **blog**: Full blog with all sections
- **about-only**: Personal site without blog features
- **minimal**: Minimal version with essential features only

## 🔧 Advanced Configuration

### Custom Shortcodes

The theme includes several useful shortcodes:

```html
{{< icon "github" >}} Social media icons {{< contact >}} Contact form {{< newsletter >}} Newsletter signup {{< list-regular-pages >}}List regular pages
```

### Taxonomies

Configure taxonomies in your `hugo.toml`:

```toml
[taxonomies]
  tag = "tags"
  category = "categories"
```

### Permalinks

Customize URL structures:

```toml
[permalinks]
  post = "/post/:year/:month/:day/:slug/"
  publications = "/publications/:slug/"
```

## 🚢 Deployment

### Static Hosting

Build your site:

```bash
npm run build:prod
```

Deploy the `public/` directory to any static hosting service:

- **Netlify**: Connect your Git repository
- **Vercel**: Import your project
- **GitHub Pages**: Use GitHub Actions
- **Cloudflare Pages**: Connect your repository

### Docker

The theme includes Docker support:

```bash
docker build -t my-site .
```

```bash
docker run -p 80:80 my-site
```

### CI/CD

Example GitHub Actions workflow:

```yaml
name: Build and Deploy
on:
    push:
        branches: [main]
jobs:
    build:
        runs-on: ubuntu-latest
        steps:
            - uses: actions/checkout@v3
            - name: Setup Hugo
              uses: peaceiris/actions-hugo@v2
            - name: Setup Node
              uses: actions/setup-node@v3
            - name: Install dependencies
              run: npm install
            - name: Build
              run: npm run build:prod
            - name: Deploy
              uses: peaceiris/actions-gh-pages@v3
```

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) before submitting PRs.

### Development Setup

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Run tests: `npm run test:e2e`
5. Submit a pull request

### Code Style

- Use ESLint and Prettier for code formatting
- Follow TypeScript best practices
- Write tests for new features
- Update documentation as needed

## 📝 License

This theme is licensed under the GPL-3.0 license - see the [LICENSE](LICENSE.txt) file for details.

## 🙏 Acknowledgments

- [Hugo](https://gohugo.io/) - Static site generator
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
- [Lucide Icons](https://lucide.dev/) - Beautiful icon library
- [Open Sans](https://fonts.google.com/specimen/Open+Sans) - Font family

## 📧 Support

- **Issues**: [GitHub Issues](https://github.com/lbenicio/aboutme-v2-theme/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lbenicio/aboutme-v2-theme/discussions)
- **Email**: hi@lbeniciod.dev

## 🔗 Resources

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Tailwind CSS v4 Guide](https://tailwindcss.com/docs/v4-beta)
- [Theme Demo](https://lbenicio.dev)

---

**Made with ❤️ by [Leonardo Benicio](https://lbenicio.dev)**
