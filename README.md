# QTIDocs

Free documentation hosting for open source projects. Put Markdown in a `qtidocs/` folder in your GitHub repo, register a subdomain, and your docs are served at `<subdomain>.qtidocs.dev`. Every push rebuilds them.

- **No build config.** The folder structure is the site structure. Frontmatter sets titles and ordering.
- **Search, dark mode and syntax highlighting** come with every site.
- **Versioned docs.** Track several branches or tags and get a version switcher.
- **PR previews.** Each pull request against your repo gets a live preview at `pr-<number>.<subdomain>.qtidocs.dev`.
- **Simple analytics.** Page views, unique visitors, top pages and referrers at `/_stats`. No cookies.

The full documentation is in [`qtidocs/`](qtidocs/) and is served by QTIDocs itself at [docs.qtidocs.dev](https://docs.qtidocs.dev).

## Quick start

1. Add a `qtidocs/` folder to your repo with at least an `index.md`:

   ```markdown
   ---
   title: Welcome
   ---

   # My Project

   Hello from QTIDocs.
   ```

2. Open a pull request against this repo that adds `sites/<subdomain>.yaml`:

   ```yaml
   subdomain: myproject
   title: My Project
   repo: github.com/you/myproject
   branch: main
   path: qtidocs/
   contact: you@example.com
   maintainers:
     - your-github-username
   ```

3. After it's merged, copy [`internal/registry/templates/deploy.yml`](internal/registry/templates/deploy.yml) into your repo's `.github/workflows/` and add the `QTIDOCS_DEPLOY_SECRET` repo secret that's emailed to your `contact` address.

For the details, see [Getting started](qtidocs/getting-started.md) and [`sites/README.md`](sites/README.md).

## How it works

```
your repo ── push / PR ──▶ deploy.yml (your Actions) ── signed POST ──▶ worker
                                                                          │
                                                       fetch tarball, extract qtidocs/,
                                                       render Markdown → static HTML
                                                                          │
reader ──▶ <subdomain>.qtidocs.dev ──▶ server ◀── routing table ◀─────────┘

this repo ── PR to sites/ ──▶ registrycheck ── merge ──▶ registrysync ──▶ platform ──▶ worker
```

QTIDocs never installs a webhook or GitHub App on your repo. Your repo's own workflow tells QTIDocs when something changed, signing each request with an HMAC key that only works for your subdomain. A reconciliation loop on the worker also re-checks each registered branch every 20 minutes, so a missed notification still gets picked up.

There are three long-running services:

| Service  | Command                        | Purpose                                                                   |
| -------- | ------------------------------ | ------------------------------------------------------------------------- |
| server   | [`cmd/server`](cmd/server)     | Serves rendered sites by `Host` header and collects analytics.            |
| platform | [`cmd/platform`](cmd/platform) | Internal API that registry CI calls to register, update and sweep sites.  |
| worker   | [`cmd/worker`](cmd/worker)     | Takes deploy and preview webhooks, queues builds and runs reconciliation. |

The rest of [`cmd/`](cmd) holds the CLIs that the GitHub Actions in [`.github/workflows/`](.github/workflows) run (`registrycheck`, `registrysync`, `registrysweep`, `disputecheck`, `disputevalidate`), plus two dev tools, `render` and `builder`.

## Development

You need Go 1.27 or newer.

```sh
go build ./...
go vet ./...
go test ./... -race
```

CI runs those three commands plus `golangci-lint` (see [`.golangci.yml`](.golangci.yml)).

### Preview a docs folder locally

`cmd/render` builds a `qtidocs/` folder into static HTML with the same pipeline production uses:

```sh
go run ./cmd/render -src qtidocs -out /tmp/qtidocs-site -title "QTIDocs"
cd /tmp/qtidocs-site && python3 -m http.server 8000
```

Then open <http://localhost:8000>. Pages use root-relative URLs, so serve the output directory as the web root. Opening the files directly with `file://` won't work.

### Run the full stack

```sh
cp .env.example .env   # fill in the secrets
docker compose up --build
```

This starts the server on `:8080`, the platform on `:8081` and the worker on `:8082`, all sharing one `data` volume for the routing table, rendered sites and analytics. See [Self-hosting](qtidocs/self-hosting.md) for every flag and environment variable.

## Repository layout

```
cmd/          entry points for the services and CI tools
internal/     everything else: build, site, markdown, sanitize, theme, nav, search,
              registry, webhook, platform, server, analytics, dispute, bot …
sites/        one <subdomain>.yaml per registered site
qtidocs/      this project's own documentation
reserved.yaml subdomains nobody can register
```

## Security

Please report vulnerabilities privately. See [SECURITY.md](SECURITY.md). If a hosted site is abusing its subdomain, use the abuse report issue template.
