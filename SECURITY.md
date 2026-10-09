# Security policy

qtidocs fetches and renders repo content from semi-trusted registrants and serves it publicly at their `*.qtidocs.dev` subdomain.

## Reporting a vulnerability

If you've found a security issue in the qtidocs platform itself (the render/sanitize pipeline, the build sandbox, the registry CI checks, the serving layer, anything in this repo's own code), not a hosted site misusing its own subdomain, see "Reporting abuse" below — please report it privately, not as a public issue:

- Preferred: open a [GitHub Security Advisory](https://github.com/quiet-terminal-interactive/qtidocs/security/advisories/new) on this repo. This keeps the report private to maintainers until a fix is ready, and GitHub Security Advisories are enabled on this repo for exactly this purpose.
- Alternative: email **hello@quietterminal.co.uk** with details.

Please include what you found, how to reproduce it, and its impact if you can. We'll acknowledge reports and keep you updated as a fix is worked on; once resolved, we'll credit you in the advisory unless you'd rather stay anonymous.

Please don't test findings (e.g. injection, sandbox escape attempts) against real registered sites other than ones you control, use a site you register yourself, or describe the issue without exploiting it against someone else's subdomain.

## Reporting abuse of a hosted site

If a `*.qtidocs.dev` subdomain is itself hosting something it shouldn't (phishing disguised as docs, spam, illegal content), that's not a security vulnerability in the platform and you should use the [abuse report issue template](.github/ISSUE_TEMPLATE/abuse-report.yml) instead, or email **hello@quietterminal.co.uk** if you'd rather not file a public issue.

## Subdomain name disputes

A trademark dispute over who a subdomain name was registered to is a separate process — see [DISPUTES.md](DISPUTES.md).
