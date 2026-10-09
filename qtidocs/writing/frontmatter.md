---
title: Frontmatter
description: Every frontmatter field a QTIDocs page supports.
order: 1
---

# Frontmatter

Every page starts with a YAML block between two `---` lines:

```markdown
---
title: Installing
description: How to install the widget factory on Linux, macOS and Windows.
order: 2
---

# Installing

…
```

The first line of the file must be exactly `---`. If a page has no frontmatter or no `title`, the whole site build fails, not just that page.

## Fields

| Field         | Type    | Default         | What it does                                                                                 |
| ------------- | ------- | --------------- | -------------------------------------------------------------------------------------------- |
| `title`       | string  | none (required) | Sets the browser tab title, the page's name in the sidebar, and its title in search results. |
| `description` | string  | none            | Becomes the page's `<meta name="description">`, which search engines and link previews use.  |
| `order`       | integer | `0`             | Sorts the page among its siblings in the sidebar. Lower numbers come first.                  |
| `nav`         | boolean | `true`          | Set to `false` to leave the page out of the auto-generated sidebar.                          |
| `author`      | string  | none            | Accepted, but not shown anywhere yet.                                                        |

Unknown fields are ignored.

### `title`

The `title` doesn't add a heading to the page itself, so start the body with your own `# Heading`. The two can differ. For example, you might use a short title for the sidebar and a longer heading on the page.

### `order`

Pages at the same level are sorted by `order`, then alphabetically by title (ignoring case). Because every page defaults to `0`, a page with a negative `order` moves above all the unordered ones.

To order a directory itself in the sidebar, set `order` in that directory's `index.md`. A directory without an `index.md` sorts as `0`.

### `nav`

A page with `nav: false` is still built, can still be found through search, and can still be linked to. It just isn't listed in the auto-generated sidebar. This is useful for changelogs, legal pages, or pages you only link to from other pages.

If you use a [`nav.yaml`](/writing/navigation), `nav` has no effect, because the file decides exactly what's in the sidebar.

## Search

Every page is added to the site's search index, including pages with `nav: false`. Search matches the page's title and the plain text of its body, and a match in the title ranks higher.
