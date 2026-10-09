---
title: Images and assets
description: Adding images, downloads and fonts with the assets/ folder.
order: 3
---

# Images and assets

Anything in `qtidocs/assets/` is copied into your site unchanged and served under `/assets/`.

```
qtidocs/
└── assets/
    ├── diagram.png      -> /assets/diagram.png
    └── fonts/
        └── Inter.woff2  -> /assets/fonts/Inter.woff2
```

## Referencing assets

In Markdown, start the path with `assets/`. It works the same from every page, however deeply the page is nested:

```markdown
![Architecture diagram](assets/diagram.png)

[Download the PDF](assets/whitepaper.pdf)
```

QTIDocs rewrites any link or image destination starting with `assets/` to `/assets/…`, so you don't need `../` paths. A path starting with `/assets/` works too.

**Versioned sites:** this rewrite doesn't add the version prefix yet. On a versioned site, assets are served at `/<version>/assets/…`, so either write the full path (`/v2/assets/diagram.png`) or see [Versioned docs](/versioning).

## Allowed file types

Assets are served with a content type based on their extension:

| Kind      | Extensions                                                 |
| --------- | ---------------------------------------------------------- |
| Images    | `.png` `.jpg` `.jpeg` `.gif` `.webp` `.avif` `.svg` `.ico` |
| Documents | `.pdf` `.txt` `.json`                                      |
| Styles    | `.css`                                                     |
| Fonts     | `.woff` `.woff2` `.ttf` `.otf`                             |

Any other file is served as `application/octet-stream`, which browsers download instead of displaying. That includes `.js` and `.html`. You can't run your own JavaScript on a QTIDocs site.

## Special files

- `assets/qtidocs.css`: if this file exists, it's loaded on every page after the default theme. See [Custom styling](/customizing).

Symlinks inside your docs folder are skipped during the build, so commit real files.
