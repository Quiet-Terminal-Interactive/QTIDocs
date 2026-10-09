---
title: Custom styling
description: Restyle your site with assets/qtidocs.css.
order: 3
---

# Custom styling

To restyle your site, add `qtidocs/assets/qtidocs.css`. It's loaded on every page after the default theme, so your rules take precedence.

## Theme variables

The easiest way to restyle is to override the theme's CSS variables:

```css
:root {
  --qtidocs-accent: #7c3aed;
  --qtidocs-font: "Inter", system-ui, sans-serif;
  --qtidocs-content-max-width: 820px;
}
```

| Variable                      | Default (light)        | Used for                               |
| ----------------------------- | ---------------------- | -------------------------------------- |
| `--qtidocs-accent`            | `#2563eb`              | Links, highlights and focus rings      |
| `--qtidocs-accent-contrast`   | `#ffffff`              | Text drawn on top of the accent colour |
| `--qtidocs-font`              | system UI stack        | Body text                              |
| `--qtidocs-mono-font`         | system monospace stack | Code                                   |
| `--qtidocs-page-bg`           | `#eef0f3`              | Background behind the content card     |
| `--qtidocs-bg`                | `#ffffff`              | Content and sidebar background         |
| `--qtidocs-fg`                | `#1f2328`              | Main text                              |
| `--qtidocs-muted-fg`          | `#57606a`              | Secondary text                         |
| `--qtidocs-border`            | `#d0d7de`              | Borders and dividers                   |
| `--qtidocs-hover-bg`          | `rgba(0,0,0,.05)`      | Hover backgrounds                      |
| `--qtidocs-code-bg`           | `#f6f8fa`              | Code block background                  |
| `--qtidocs-nav-width`         | `280px`                | Sidebar width                          |
| `--qtidocs-content-max-width` | `760px`                | Maximum width of the page content      |

## Dark mode

Readers can switch between light and dark with the button in the sidebar header, and their choice is remembered. If they haven't chosen, the site follows their system setting. To change the dark colours, override the variables for both cases:

```css
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --qtidocs-accent: #c4b5fd;
  }
}
:root[data-theme="dark"] {
  --qtidocs-accent: #c4b5fd;
}
```

## Targeting elements

The theme's own elements use `qtidocs-` class names, such as `.qtidocs-nav`, `.qtidocs-content`, `.qtidocs-nav-home` and `.qtidocs-search`. Page content is inside `.qtidocs-content article`.

You can also style your own classes from inline HTML in your pages:

```html
<div class="callout">Remember to restart the server.</div>
```

```css
.callout {
  border-left: 4px solid var(--qtidocs-accent);
  padding: .75rem 1rem;
  background: var(--qtidocs-code-bg);
}
```

## Fonts and images

`url()` can point to files in your own `assets/` folder:

```css
@font-face {
  font-family: "Inter";
  src: url("/assets/fonts/Inter.woff2") format("woff2");
}
```

## Restrictions

`qtidocs.css` is checked during every build, and the build **fails** if the file:

- uses `@import`;
- uses `url()` with an absolute or external URL (`https://…` or `//…`). Use paths into `/assets/` instead. `data:` URLs are allowed;
- isn't valid CSS.

This is why Google Fonts and other CDN stylesheets can't be loaded. Download the font files into `assets/` instead.
