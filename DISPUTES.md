# Subdomain name disputes

This is the process for contesting who a `*.qtidocs.dev` subdomain is registered to, on trademark grounds. It is **not** a way to reclaim a name you simply want, or think you'd put to better use — registration is otherwise first-come-first-served, and that stands unless this process says otherwise.

## Who can file, and on what grounds

Anyone can file a dispute, but only on trademark-infringement grounds: you hold a trademark on a name, and a registered subdomain using that name conflicts with it. "I had the idea first," "I want this name more," or similar are not valid grounds and will be closed without review.

Disputes over a hosted site's *content* (phishing, spam, illegal content) are a different process — see [SECURITY.md](SECURITY.md#reporting-abuse-of-a-hosted-site) instead.

## Filing a dispute

Open an issue on this repo using the **[Subdomain name dispute](.github/ISSUE_TEMPLATE/subdomain-dispute.md)** template. You'll need:

- The subdomain being disputed, and its current registrant/repo.
- The trademark being asserted, with evidence of your rights to it (registration number, a link to a trademark office record, or similar).
- Why the current use conflicts with that mark.

Filing the issue starts a 14-day opposition window, starting the day you file. Fill in the template's `Deadline:` line with the exact date 14 calendar days out, the issue is validated automatically the moment you open it, and closed immediately (with a comment asking you to reopen it with the correct date) if that line isn't exactly right.

## What happens during the opposition window

The current registrant can respond directly in the issue to oppose the dispute. Whether or not they do, resolving the dispute is at the discretion of a Quiet-Terminal-Interactive org member, they weigh the evidence and any opposition, and decide:

- **Dispute rejected**: the subdomain stays with its current registrant. Nothing further happens.
- **Dispute accepted**: the subdomain is reassigned away from its current registrant, whether or not they opposed it.

An automated check comments as the 14-day deadline approaches and flags the issue for a maintainer once it's passed, so a dispute doesn't just quietly expire unresolved.

## Appealing an outcome

If you lost a subdomain to a dispute and think that decision was wrong, you have 31 days from the date you lost it to appeal. Open an issue using the **[Dispute outcome appeal](.github/ISSUE_TEMPLATE/dispute-outcome-appeal.md)** template, linking the original dispute issue. This is for contesting the outcome of an already-resolved dispute, not a way to reopen the original trademark question from scratch on the same grounds.

Fill in the template's `Reassigned:` line with the date the subdomain was reassigned away from you, and `Deadline:` with the date exactly 31 calendar days after that. As with the initial filing, this is validated automatically the moment you open the issue.

An appeal is decided the same way an initial dispute is: at the discretion of a Quiet-Terminal-Interactive org member, who can reverse the reassignment or let it stand.

## Why the dates matter

The `Deadline:`/`Reassigned:` lines aren't just for readers, an automated check parses them to post reminders as a deadline approaches and to flag the issue for a maintainer once it's passed. Keep them in the exact `Label: YYYY-MM-DD` format the template shows, with nothing else on that line, or the issue gets closed automatically and you'll need to reopen it with the dates corrected.
