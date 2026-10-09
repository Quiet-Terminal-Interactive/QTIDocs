---
title: Self-hosting
description: Run your own QTIDocs platform.
order: 8
---

# Self-hosting

QTIDocs is three Go services that share a data directory. The repo includes a `Dockerfile` and a `docker-compose.yml` that run all three.

```sh
git clone https://github.com/quiet-terminal-interactive/qtidocs
cd qtidocs
cp .env.example .env    # fill in the values below
docker compose up --build
```

## Services

| Service    | Port | Role                                                                                                                                                                            |
| ---------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `server`   | 8080 | Public. Serves every site, choosing it from the `Host` header (`<subdomain>.<domain>`), and records analytics.                                                                  |
| `platform` | 8081 | Internal. Registry CI calls it to register sites, apply updates and run the nightly sweep.                                                                                      |
| `worker`   | 8082 | Takes deploy and preview webhooks from users' repos, queues and runs builds, and checks branches for changes. `/deploy` and `/preview/*` must be reachable from GitHub Actions. |

All three read the same routing table (`routing.json`), which maps each subdomain to its repo and rendered output.

## Environment

| Variable                  | Used by          | Purpose                                                                            |
| ------------------------- | ---------------- | ---------------------------------------------------------------------------------- |
| `QTIDOCS_GITHUB_TOKEN`    | worker           | Fetches tarballs and checks refs. Optional, but it raises GitHub's rate limit.     |
| `QTIDOCS_BOT_TOKEN`       | worker           | Opens issues on repos when their build fails. If it's unset, no issues are opened. |
| `QTIDOCS_PLATFORM_SECRET` | platform         | Bearer token that registry CI uses to call the platform.                           |
| `QTIDOCS_WORKER_SECRET`   | worker, platform | Bearer token the platform uses to call the worker's internal endpoints.            |
| `QTIDOCS_SMTP_HOST`       | platform         | SMTP relay that emails each new site's deploy secret to its `contact`. Required.   |
| `QTIDOCS_SMTP_PORT`       | platform         | Relay port, `587` by default. `465` uses implicit TLS; other ports need STARTTLS.  |
| `QTIDOCS_SMTP_USERNAME`   | platform         | Relay username. Set together with the password, or leave both empty.               |
| `QTIDOCS_SMTP_PASSWORD`   | platform         | Relay password.                                                                    |
| `QTIDOCS_SMTP_FROM`       | platform         | From address on deploy-secret emails, e.g. `noreply@yourdomain`. Required.         |

## Flags

### `server`

| Flag                   | Default | Meaning                                                                                      |
| ---------------------- | ------- | -------------------------------------------------------------------------------------------- |
| `-routing`             | none    | Path to the routing table. Can't be combined with `-root`.                                   |
| `-root`                | none    | Serve every subdirectory as a site, without a routing table. Useful for local testing.       |
| `-addr`                | `:8080` | Listen address.                                                                              |
| `-analytics`           | none    | Directory where analytics are stored. If it's unset, analytics and `/_stats` are turned off. |
| `-analytics-retention` | `720h`  | How long raw page-view events are kept.                                                      |

### `worker`

| Flag                   | Default                  | Meaning                                                              |
| ---------------------- | ------------------------ | -------------------------------------------------------------------- |
| `-routing`             | none                     | Path to the routing table.                                           |
| `-sites-root`          | none                     | Directory that rendered sites are written to.                        |
| `-addr`                | `:8082`                  | Listen address.                                                      |
| `-secret`              | `$QTIDOCS_WORKER_SECRET` | Secret for the internal endpoints.                                   |
| `-github-token`        | `$QTIDOCS_GITHUB_TOKEN`  | GitHub API token.                                                    |
| `-bot-token`           | `$QTIDOCS_BOT_TOKEN`     | Token for opening build-failure issues.                              |
| `-reconcile-interval`  | `20m`                    | How often to check each branch tip against the last deployed commit. |
| `-max-extracted-bytes` | 50 MB                    | Build limit.                                                         |
| `-max-files`           | `5000`                   | Build limit.                                                         |
| `-max-build-time`      | `2m`                     | Build limit.                                                         |
| `-max-output-bytes`    | 100 MB                   | Build limit.                                                         |

### `platform`

| Flag                   | Default                    | Meaning                                                                         |
| ---------------------- | -------------------------- | ------------------------------------------------------------------------------- |
| `-routing`             | none                       | Path to the routing table.                                                      |
| `-addr`                | `:8081`                    | Listen address.                                                                 |
| `-secret`              | `$QTIDOCS_PLATFORM_SECRET` | Secret for registration requests.                                               |
| `-worker-url`          | none                       | The worker's `/internal/build` endpoint. If it's unset, builds are only logged. |
| `-worker-teardown-url` | none                       | The worker's `/internal/teardown` endpoint. Required when `-worker-url` is set. |
| `-worker-secret`       | `$QTIDOCS_WORKER_SECRET`   | Secret used to call the worker.                                                 |
| `-smtp-host`           | `$QTIDOCS_SMTP_HOST`       | SMTP relay for deploy-secret emails. Required.                                  |
| `-smtp-port`           | `$QTIDOCS_SMTP_PORT`, 587  | Relay port.                                                                     |
| `-smtp-username`       | `$QTIDOCS_SMTP_USERNAME`   | Relay username.                                                                 |
| `-smtp-password`       | `$QTIDOCS_SMTP_PASSWORD`   | Relay password.                                                                 |
| `-smtp-from`           | `$QTIDOCS_SMTP_FROM`       | From address. Required.                                                         |

When a subdomain is registered for the first time, the platform generates its deploy secret and emails it, with setup instructions, to the `contact` address in the site's registry entry. The secret is never returned over HTTP or printed in CI logs. If the email can't be sent, the registration fails and nothing is stored, so rerunning the registry deploy workflow sends a fresh secret. Re-registrations never resend the existing secret, even if `contact` has changed.

## DNS and TLS

Point a wildcard record (`*.yourdomain`) at `server`, and put a TLS-terminating reverse proxy in front of it with a wildcard certificate. Preview sites use two-level names (`pr-42.acme.yourdomain`), so they need their own wildcard per site, or a proxy that issues certificates on demand.

The worker's public endpoints (`/deploy`, `/preview/deploy` and `/preview/teardown`) need to be reachable at the URL in `deploy.yml`, which is `api.qtidocs.dev` upstream. Change it in `internal/registry/templates/deploy.yml` for your instance.

## Registry automation

Registration runs through GitHub Actions on your fork of this repo:

- `registry-check.yml` validates PRs that touch `sites/`.
- `registry-deploy.yml` calls the platform after a merge.
- `registry-sweep.yml` runs every night and removes sites whose files were deleted.

The deploy and sweep workflows authenticate with a `QTIDOCS_PLATFORM_SECRET` repository secret. The platform URL is hard-coded as `https://api.qtidocs.dev/internal/…`, so change it in both workflow files for your instance.
