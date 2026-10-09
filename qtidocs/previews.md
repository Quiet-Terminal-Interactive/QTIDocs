---
title: PR previews
description: Every pull request gets a live preview of its docs.
order: 5
---

# PR previews

Once you've added the [deploy workflow](/getting-started) to your repo, each pull request gets its own live copy of your docs at:

```
pr-<number>.<subdomain>.qtidocs.dev
```

For example, PR #42 on `acme` is previewed at `pr-42.acme.qtidocs.dev`.

- The preview is built from the docs folder on the PR's head commit.
- It's rebuilt each time new commits are pushed to the PR.
- It's deleted as soon as the PR is closed, whether or not it was merged.

Names that match `pr-<number>` can't be registered as subdomains, so previews never clash with a real site.

## Requirements

- The deploy workflow must be in your repo, with the `pull_request` trigger it ships with.
- The `QTIDOCS_DEPLOY_SECRET` secret must be available to the workflow run. GitHub doesn't give repository secrets to workflows triggered by PRs **from forks**, so those PRs won't get a preview. PRs from branches in your own repo work normally.
