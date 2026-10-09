---
title: Analytics
description: Privacy-friendly page view stats at /_stats.
order: 6
---

# Analytics

Every site has a stats page at `/_stats`, for example `acme.qtidocs.dev/_stats`. It shows:

- page views and unique visitors;
- your most-viewed pages;
- top referrers;

for today, the last 7 days, the last 30 days or all time.

## What's collected

When a page loads, a small first-party script sends the page path and the referrer to your own subdomain. The server stores:

- the path;
- the referrer;
- a timestamp;
- a visitor hash.

The visitor hash is a one-way hash of the visitor's IP address, user agent and the current date. It's only used to count unique visitors per day, and because the date is part of it, the same visitor can't be followed across days. QTIDocs doesn't set cookies, use third-party trackers, or store IP addresses.

Raw events are deleted after 30 days. Daily totals are kept so that all-time stats still work.

## Who can see it

**`/_stats` is public.** Anyone who knows the URL can see your site's traffic numbers. If that's a problem for you, open an issue.
