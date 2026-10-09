---
title: Versioned docs
description: Publish docs for several releases, with a version switcher.
order: 4
---

# Versioned docs

If your project has several supported releases, QTIDocs can publish a copy of your docs for each one. Every page then gets a version switcher.

## Setting it up

List the refs to track under `versions` in your registry entry (`sites/<subdomain>.yaml`):

```yaml
subdomain: acme
repo: github.com/acme-inc/widget-factory
branch: main
path: qtidocs/
contact: docs@acme.example
maintainers:
  - acme-jane
versions:
  - name: v1
    ref: v1.0.0
  - name: v2
    ref: v2.0.0
  - name: latest
    ref: main
default_version: v2
```

- `name` is the URL prefix and the label in the switcher. Each version is served at `/<name>/…`, such as `acme.qtidocs.dev/v2/getting-started`.
- `ref` can be any branch, tag or commit. Each version is built from that ref's own docs folder.
- `default_version` sets where `acme.qtidocs.dev/` redirects. If you leave it out, the first version is used.

Version names must be unique, and `path` must exist at every listed ref. The registry check verifies both.

## How it behaves

- The version switcher keeps the reader on the same page in the other version. If that page doesn't exist there, they get a 404.
- Each version has its own sidebar, `nav.yaml`, `qtidocs.css` and search index.
- A push rebuilds **every** version. A push doesn't say which version it belongs to, so QTIDocs rebuilds them all.

## Known limitations

- **Links that start with `/` don't get the version prefix.** A Markdown link to `/getting-started` goes to the unversioned path, which doesn't exist on a versioned site. For now, include the version in the link (`/v2/getting-started`). Sidebar links are prefixed automatically.
- **`assets/` links aren't prefixed either.** `![x](assets/x.png)` becomes `/assets/x.png`, but the file is served at `/<version>/assets/x.png`. Write the full versioned path for now. `qtidocs.css` is loaded from the right place automatically.

## Adding or removing a version

Edit `versions` in your registry entry with a PR. You must be a listed maintainer. The site is rebuilt with the new set of versions after the merge.
