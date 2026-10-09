---
title: Limits and troubleshooting
description: Build limits, and what to do when a build fails.
order: 7
---

# Limits and troubleshooting

## Build limits

Each build fetches your repo as a tarball, extracts your docs folder, and renders it. To keep the platform fair, every build has these limits:

| Limit                             | Value     | Notes                                                                          |
| --------------------------------- | --------- | ------------------------------------------------------------------------------ |
| Files in the repo tarball         | 5,000     | Every file in the **whole repo** at that ref counts, not just the docs folder. |
| Size of the extracted docs folder | 50 MB     | Only files under `path` count.                                                 |
| Size of the rendered output       | 100 MB    |                                                                                |
| Build time                        | 2 minutes | Fetching and rendering combined.                                               |

If a large monorepo hits the file limit, you can keep your docs in a separate, smaller repo and register that repo instead.

## When a build fails

If a build fails, the previous version of your site stays online, and the **Quiet-Terminal-Bot** opens an issue on your repo with the error.

Common causes:

| Error mentions                        | Fix                                                                                                             |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `frontmatter: title is required`      | A `.md` file has no frontmatter, or no `title`. The error names the file. The first line must be exactly `---`. |
| `frontmatter: unterminated --- block` | The frontmatter's closing `---` is missing.                                                                     |
| `frontmatter: yaml: …`                | The frontmatter isn't valid YAML. Titles that contain a `:` usually need quotes.                                |
| `nav.yaml: …`                         | `nav.yaml` isn't valid YAML, or isn't a list at the top level.                                                  |
| `qtidocs.css: …`                      | Your CSS uses `@import` or an external `url()`. See [Custom styling](/customizing).                             |
| `more than 5000 entries`              | The repo has too many files. See the build limits above.                                                        |
| `exceeds the configured size limit`   | The docs folder is over 50 MB. Move large binaries out of it.                                                   |
| `fetching tarball … 404`              | The repo is private, was renamed, or the ref no longer exists.                                                  |

You can reproduce most of these locally with `cmd/render`. See [Writing docs](/writing).

## My push didn't update the site

1. Check that the **qtidocs deploy** workflow ran in your repo's Actions tab, and that it succeeded.
2. Check that the branch in `deploy.yml` matches `branch` in your registry entry.
3. Check that the `QTIDOCS_DEPLOY_SECRET` secret is set and matches the one emailed to your entry's `contact` address when the subdomain was registered (subject "Your QTIDocs deploy secret for …"; check spam too). If it's wrong, the request is rejected. The secret is sent only once, so if you've lost the email, open an issue.

Even without a notification, QTIDocs re-checks your branch about every 20 minutes and rebuilds if it has changed.

## Still stuck?

Open an issue using the bug report template on [GitHub](https://github.com/quiet-terminal-interactive/qtidocs/issues).
