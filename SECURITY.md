# Security Policy

## Supported versions

`again` is in beta. Security fixes are made on `master` and shipped in the next release. Only the latest release is supported.

## Reporting a vulnerability

**Don't report vulnerabilities in public issues, pull requests or comments.**

Report privately through [GitHub private vulnerability reporting](https://github.com/msaeedsaeedi/again/security/advisories/new) (the **Security** tab, then **Report a vulnerability**).

Please include:

- The affected version or commit, and your operating system
- Steps to reproduce, ideally a minimal proof of concept
- The impact: what an attacker needs, and what they gain
- A suggested fix, if you have one

## What to expect

This project is maintained by one person, so these are targets rather than guarantees:

- Acknowledgement within 7 days
- An initial assessment within 14 days
- Progress updates until the report is resolved

We follow coordinated disclosure. Please give us up to 90 days to release a fix before disclosing publicly. Accepted reports are published as GitHub security advisories, with credit if you want it.

## Scope

`again` runs the commands you give it, with your permissions. That is its purpose and isn't a vulnerability.

Examples that are in scope:

- Running something other than the command the user asked for
- Processes that survive cancellation in a way that has a security impact
- Command output that breaks out of an output format or manipulates the terminal through the TUI
- Weaknesses in the build or release process, or in repository workflows

## Research guidelines

Test only on systems and forks you own. Don't access other people's data, disrupt services, or run proof-of-concept workflows against this repository.
