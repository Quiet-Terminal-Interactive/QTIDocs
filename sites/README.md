# sites/

One file per registered subdomain: `<subdomain>.yaml`. To register a new
site at `<subdomain>.qtidocs.dev`, open a pull request adding a file here.

```yaml
subdomain: acme
title: Acme Docs
repo: github.com/acme-inc/widget-factory
branch: main
path: qtidocs/
contact: docs@acme.example
maintainers:
  - acme-jane
  - acme-bob
```

- `subdomain` must match the filename (`acme.yaml` → `subdomain: acme`), be unique, and not be on the reserved list (`../reserved.yaml`).
- `title` (optional) is shown in your site's nav header as `<title> | QTIDocs`; omit it and the nav header just shows `QTIDocs`.
- `repo` is the GitHub repo containing your `qtidocs/` folder (`github.com/<owner>/<name>`). The PR author needs write access to it, unless they're registering on someone else's behalf as a member of the `quiet-terminal-interactive` org.
- `path` is where your `qtidocs/` folder lives in that repo, and must exist at `branch` at merge time.
- `contact` is the email address your `QTIDOCS_DEPLOY_SECRET` is sent to when the subdomain is first registered. This file is public, so use an address you don't mind publishing. Changing `contact` or `repo` later replaces the secret: a new one is emailed to the new `contact`, the old one stops working, and any open PR previews are torn down.
- `maintainers` is the list of GitHub usernames allowed to submit future PRs *editing* this file, not just whoever had write access at registration time.

Optional, for versioned docs: list the branches/tags to track under `versions`, each producing its own `/<name>/`-prefixed copy of your site (e.g. `acme.qtidocs.dev/v2/...`), built from that ref's own `qtidocs/` folder.

```yaml
versions:
  - name: v1
    ref: v1.0.0
  - name: v2
    ref: v2.0.0
default_version: v2
```

`default_version` is which one is served at your site's root (`acme.qtidocs.dev/` redirects to `/v2/`); if you leave it unset, the first entry in `versions` is used. Every rendered page of a versioned site carries a version switcher in its nav, letting readers jump to the same page in another tracked version. A push to any branch/tag of your repo rebuilds every tracked version at once (there's no way to tell from a push alone which version, if any, it corresponds to), so `path` must exist at every listed ref, not just `branch`.

Once you've added the workflow template below, opening a pull request against your own repo (not this one) automatically gets a live preview at `pr-<number>.<subdomain>.qtidocs.dev`, built from that PR's own `qtidocs/` folder and rebuilt on every new commit pushed to it. The preview is torn down immediately when the PR is closed, merged or not, it's never left for the nightly cleanup cron. `pr-<number>` subdomains are reserved platform-wide, so this can never collide with a real registration.

Every PR touching this directory is checked automatically by `.github/workflows/registry-check.yml` (`cmd/registrycheck`), but a human maintainer still has to merge it.

Once merged, `.github/workflows/registry-deploy.yml` (`cmd/registrysync`) tells the platform about the change, which adds/updates your subdomain's routing-table entry and triggers an initial build. Add the Actions workflow template (`internal/registry/templates/deploy.yml`) to your own repo so future pushes trigger a rebuild too, and pull requests get the preview deploys described above, it needs a `QTIDOCS_DEPLOY_SECRET` repo secret, generated the first time your subdomain is registered and emailed to `contact` along with setup instructions.

To edit an existing entry, open a PR changing its file; the check verifies you're listed in its *current* `maintainers` field. To unregister, open a PR deleting the file, teardown happens via the nightly cleanup cron, not immediately on merge.

If someone else holds a trademark on a name a registered subdomain is using, see [DISPUTES.md](../DISPUTES.md) for how to contest it — this is a separate, trademark-only process, not a way to reclaim a name you just want. To report a site misusing its subdomain (phishing, spam, illegal content), see [SECURITY.md](../SECURITY.md#reporting-abuse-of-a-hosted-site).
