---
title: Navigation
description: How the sidebar is generated and how to replace it with nav.yaml.
order: 4
---

# Navigation

## The automatic sidebar

By default, the sidebar mirrors your folder structure:

- Each directory becomes a section, and each page becomes an entry.
- A section uses the `title` of its `index.md`, and clicking it opens that page. If a directory has no `index.md`, its name is used instead (`getting-started` becomes "Getting Started"), and the section isn't a link.
- Entries are sorted by [`order`](/writing/frontmatter), then alphabetically by title.
- Pages with `nav: false` are left out.
- The home page (`qtidocs/index.md`) isn't listed. Clicking the site title at the top of the sidebar opens it.

For most sites, setting `order` in frontmatter is all you need.

## A custom sidebar with `nav.yaml`

For full control, add `qtidocs/nav.yaml`. When this file exists, it replaces the automatic sidebar entirely:

```yaml
- path: getting-started
- title: Guides
  items:
    - path: guides/install
    - path: guides/configure
      title: Configuration
- title: Reference
  path: reference
  items:
    - path: reference/cli
    - path: reference/api
- title: Changelog
  path: changelog
```

Each entry can have these keys:

| Key     | Meaning                                                                                      |
| ------- | -------------------------------------------------------------------------------------------- |
| `path`  | The page's URL path, **without a leading `/`**. Omit it to make a heading that isn't a link. |
| `title` | The text shown in the sidebar. Omit it to use the page's frontmatter `title`.                |
| `items` | Nested entries shown under this one.                                                         |

Things to know:

- Write paths without a leading slash: `guides/install`, not `/guides/install`.
- `nav.yaml` doesn't check that its paths exist. A typo builds fine and produces a broken link.
- Pages left out of `nav.yaml` are still built and searchable.
- `sidebar.yaml` works as another name for the same file. If both exist, `nav.yaml` is used.
- On [versioned sites](/versioning), each version reads its own `nav.yaml` from its own ref, and the paths get the version prefix automatically.
