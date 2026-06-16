# Security Policy

Thank you for helping keep this project and its users safe. This document describes how to report vulnerabilities and our disclosure policy.

## Supported Versions

The following versions of `aboutme-v2-theme` are currently supported with security updates:

| Version | Supported |
| ------: | :-------: |
|   0.1.x |    Yes    |
| < 0.1.0 |    No     |

## Reporting a Vulnerability

Please report security issues responsibly and privately.

- Preferred: Open a private report via GitHub Security Advisories — [Create a private advisory](https://github.com/lbenicio/lbenicio/aboutme-v2-theme/security/advisories/new)
- Alternative: If you prefer email, use the contact email published on the site (configured via `NEXT_PUBLIC_CONTACT_EMAIL`). For encrypted mail, you can use the PGP key located at `public/pgp/public.asc` in this repository (served at `/pgp/public.asc` when the site is running).

When reporting, please include:

- A clear description of the issue and potential impact
- Steps to reproduce (PoC), affected URLs/components, and any logs
- Version information (app version, commit SHA), environment, and configuration details if relevant
- Your preferred contact method for follow‑up

Please avoid sharing sensitive user data. If the issue involves data exposure, redact or synthesize any personal information in your report.

## Disclosure Policy

- We will acknowledge receipt within 2 business days.
- We will provide an initial assessment or request for more information within 5 business days.
- We aim to release a fix and coordinated disclosure as soon as practical, prioritizing critical issues. Timelines vary by severity and complexity.
- We will credit reporters who request acknowledgment, unless you prefer to remain anonymous.

## Scope

In scope:

- Code and configurations contained in this repository
- Official container images and deployment artifacts produced by this project

Out of scope:

- Third‑party services, dependencies, or infrastructure not maintained by this project
- Social engineering, SPF/DMARC/DKIM or email spoofing unrelated to project code

## Safe Harbor

We support good-faith research. If you follow this policy and:

- Avoid privacy violations, data destruction, or service disruption
- Do not access or modify data without authorization
- Make a good‑faith effort to report promptly and privately

we will not pursue legal action against you. If in doubt, ask first via a private advisory report.
