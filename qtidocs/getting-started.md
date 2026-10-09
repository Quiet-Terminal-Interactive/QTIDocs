---
title: Getting started
description: Register a subdomain and publish your first QTIDocs site.
order: 1
---

# Getting started

Publishing a site takes three steps: add a docs folder to your repo, register a subdomain, and add a workflow so pushes trigger rebuilds.

## 1. Add a `qtidocs/` folder

Create a `qtidocs/` folder in your repo with at least one Markdown page. Every page needs a frontmatter block with a `title`:

```markdown
---
title: Welcome
---

# My Project

Hello from QTIDocs.
```

Save this as `qtidocs/index.md`. It becomes your site's home page. Push it to the branch you want to publish from, usually `main`.

The folder can live anywhere in the repo, such as `docs/qtidocs/`. You'll tell QTIDocs where it is in the next step.

**Tip:** you can preview the site locally with `cmd/render` before you register. See [Writing docs](/writing).

## 2. Register a subdomain

Open a pull request against [quiet-terminal-interactive/qtidocs](https://github.com/quiet-terminal-interactive/qtidocs) that adds `sites/<subdomain>.yaml`:

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

| Field             | Required | Meaning                                                                                                                                     |
| ----------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `subdomain`       | yes      | Your site's address, `<subdomain>.qtidocs.dev`. Use lowercase letters, digits and hyphens, up to 63 characters. It must match the filename. |
| `title`           | no       | Shown in the nav header as `<title> \| QTIDocs`. If you leave it out, the header shows just `QTIDocs`.                                      |
| `repo`            | yes      | The GitHub repo containing your docs, as `github.com/<owner>/<name>`.                                                                       |
| `branch`          | yes      | The branch to build from.                                                                                                                   |
| `path`            | yes      | Where the docs folder is in the repo. It must exist on `branch` when the PR is merged.                                                      |
| `contact`         | yes      | Email address your deploy secret is sent to. This file is public, so use an address you're happy to publish.                                |
| `maintainers`     | yes      | GitHub usernames allowed to edit or delete this entry later.                                                                                |
| `versions`        | no       | Branches or tags to publish as separate versions. See [Versioned docs](/versioning).                                                        |
| `default_version` | no       | The version served at your site's root.                                                                                                     |

A CI check validates the PR automatically. It confirms that:

- the subdomain isn't taken or reserved (one- and two-letter names are reserved, as are the names in [`reserved.yaml`](https://github.com/quiet-terminal-interactive/qtidocs/blob/main/reserved.yaml));
- you have write access to `repo`;
- `path` exists on `branch` and on every ref listed in `versions`.

A human maintainer still reviews and merges the PR. After the merge, your site gets its first build.

## 3. Add the deploy workflow

QTIDocs doesn't install a webhook or GitHub App on your repo. Instead, your repo notifies QTIDocs itself. Copy [`deploy.yml`](https://github.com/quiet-terminal-interactive/qtidocs/blob/main/internal/registry/templates/deploy.yml) into your repo's `.github/workflows/` folder.

If you publish from a branch other than `main`, change the branch under `on.push.branches` to match `branch` in your registry entry. Otherwise, use the file as it is.

The workflow needs a repo secret named `QTIDOCS_DEPLOY_SECRET`. It's generated when your subdomain is registered and emailed to your entry's `contact` address, together with these instructions. It's sent only once and never appears in CI logs. Add it under **Settings -> Secrets and variables -> Actions** in your repo, or run `gh secret set QTIDOCS_DEPLOY_SECRET --repo <owner>/<name>`.

From then on:

- every push to your branch rebuilds the site;
- every pull request gets a [preview](/previews).

If a notification is missed, QTIDocs re-checks your branch about every 20 minutes and rebuilds if it has changed.

## Changing or removing your site

- **To edit** your entry, for example to change `branch` or add maintainers, open a PR that changes `sites/<subdomain>.yaml`. You must be listed in the entry's current `maintainers`. The subdomain can't be renamed. To move to a new name, unregister and register again.
- **To unregister**, open a PR that deletes the file. The site is taken down by the nightly cleanup job, not immediately on merge.

## Next steps

- [Frontmatter](/writing/frontmatter) lists every field a page supports.
- [Navigation](/writing/navigation) explains how to control your sidebar.
