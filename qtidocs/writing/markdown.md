---
title: Markdown
description: Supported Markdown syntax, inline HTML and code highlighting.
order: 2
---

# Markdown

Pages are rendered with [GitHub Flavored Markdown](https://github.github.com/gfm/), so if it looks right on GitHub it will look much the same here.

## Supported syntax

All of CommonMark, plus the GFM extensions:

- tables;
- ~~strikethrough~~;
- task lists (`- [x] done`);
- bare URLs that link automatically, like https://qtidocs.dev.

| Syntax          | Example                               |
| --------------- | ------------------------------------- |
| Inline code     | `` `go test ./...` ``                 |
| Bold and italic | `**bold**` and `*italic*`             |
| Links           | `[Getting started](/getting-started)` |
| Images          | `![Logo](assets/logo.png)`            |

## Code blocks

Fenced code blocks with a language are syntax highlighted, and the colours follow the reader's light or dark theme:

````markdown
```go
func main() {
    fmt.Println("hello")
}
```
````

The highlighter recognises hundreds of languages, using their usual names such as `go`, `rust`, `python`, `ts`, `yaml`, `sh` and `diff`. A block without a language is shown as plain monospaced text.

## Links between pages

Link to other pages by their URL path, starting with `/`:

```markdown
See [Frontmatter](/writing/frontmatter).
```

Leave off the `.md` extension. Relative links like `frontmatter.md` won't work, because the rendered site has no `.md` files.

Headings don't get `id` attributes, so links to a heading on a page (`/page#section`) won't jump to it. Link to the page itself instead.

External links open normally and get `rel="nofollow"`.

## Inline HTML

You can use HTML in your Markdown, but it's sanitized against an allowlist. These elements are kept:

- p
- br
- hr
- b
- strong
- i
- em
- u
- s
- del
- ins
- sub
- sup
- mark
- small
- kbd
- details
- summary
- div
- span
- ul
- ol
- li
- dl
- dt
- dd
- table
- thead
- tbody
- tfoot
- tr
- th
- td
- caption
- h1-h6
- pre
- code
- a
- img

These attributes are kept:

- `class` and `id` on any element;
- `href` on links, which must be `http`, `https`, `mailto` or a relative URL;
- `src`, `alt`, `width`, `height` and `title` on images.

Everything else is removed. That includes `<script>`, `<iframe>`, `<style>`, `<form>`, event handlers like `onclick`, and `style` attributes. When an element isn't allowed, its tags are removed but its text is kept.

A collapsible section is a useful example of HTML that is allowed:

```html
<details>
<summary>Show the full config</summary>

…long content…

</details>
```

### Blockquotes

`<blockquote>` isn't on the allowlist yet, so a `> quote` is shown as a normal paragraph. For callouts, use a bold label such as `**Note:**`, or use a `<div class="…">` and style it in your [`qtidocs.css`](/customizing).

## Images

Put images in your `assets/` folder and reference them as `assets/<file>`. See [Images and assets](/writing/assets).

Images from other websites are blocked by the site's Content Security Policy, which only allows resources from your own subdomain. Download any image you need into `assets/`.
