---
title: Writing docs
description: How a qtidocs/ folder turns into a site.
order: 2
---

# Writing docs

A QTIDocs site is a folder of Markdown files. The folder structure becomes the URL structure and the sidebar.

```
qtidocs/
├── index.md              -> /
├── getting-started.md    -> /getting-started
├── guides/
│   ├── index.md          -> /guides
│   ├── install.md        -> /guides/install
│   └── configure.md      -> /guides/configure
├── assets/               -> /assets/… (copied as-is, never rendered)
│   ├── logo.png
│   └── qtidocs.css       -> optional style overrides
└── nav.yaml              -> optional explicit sidebar
```

The rules are:

- **Every `.md` file is a page.** `foo/bar.md` is served at `/foo/bar`.
- **`index.md` is the page for its directory.** `guides/index.md` is served at `/guides`, and the top-level `index.md` is your home page.
- **Every page must start with frontmatter that includes a `title`.** A page without one fails the whole build. See [Frontmatter](/writing/frontmatter).
- **`assets/` is for static files.** It's copied into the site unchanged, and Markdown files inside it are not rendered. See [Images and assets](/writing/assets).
- **Other files are ignored.** That includes non-Markdown files outside `assets/`.

## In this section

- [Frontmatter](/writing/frontmatter): the fields every page can set.
- [Markdown](/writing/markdown): supported syntax, HTML and code highlighting.
- [Images and assets](/writing/assets): adding images, downloads and fonts.
- [Navigation](/writing/navigation): how the sidebar is built and how to override it.

## Previewing locally

To see your site before you push, clone the QTIDocs repo and run the renderer on your folder. It uses the same pipeline as production:

```sh
git clone https://github.com/quiet-terminal-interactive/qtidocs
cd qtidocs
go run ./cmd/render -src /path/to/your/repo/qtidocs -out /tmp/my-site -title "My Project"
cd /tmp/my-site && python3 -m http.server 8000
```

Then open <http://localhost:8000>. Serve the output directory as the web root, because opening the HTML files directly won't load styles or search.

If the renderer reports an error, such as a page missing its `title` or an invalid `qtidocs.css`, the real build would fail in the same way.

You can also open a pull request on your repo to get a hosted [preview](/previews).
