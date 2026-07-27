# Security Policy

We take the security of `sql_exporter` seriously. Because this project manages database connections, handles sensitive
Data Source Names (DSNs), and exposes operational metrics, maintaining security and privacy is critical.

Given the recent trends in supply chain attacks as of 2026, we recommend that users also take extra care to verify the
authenticity of any release binaries or container images they download.

On our end, we do as much as possible to track and prevent vulnerabilities, but we cannot guarantee that all issues
will be discovered before a release. Therefore, we strongly encourage users to verify the integrity of any downloaded
artifacts and to follow best practices for secure deployment. What was considered secure yesterday may not be secure
today, so please stay vigilant and keep your deployments up to date.

## Supported Versions

Only the latest release of `sql_exporter` receives active security patches and dependency updates. We strongly
encourage all users to run the most recent release.

| Version  | Supported          |
| -------- | ------------------ |
| Latest   | :white_check_mark: |
| < Latest | :x:                |


## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

If you discover a security vulnerability in `sql_exporter`, please report it privately:

1. **GitHub Private Vulnerability Reporting (Preferred):**
   Navigate to the **Security** tab of this repository, click **Report a vulnerability**, and submit your report.
2. **Email:**
   Send an email with the subject `[SECURITY] sql_exporter` to `oss [at] szyubin [dot] me`.

### What to Include in Your Report

To help us assess and resolve the vulnerability effectively, please include:

- **Description:** A detailed explanation of the issue and its potential impact (e.g., credential exposure, SQL
  injection vector, unexpected metric leakage, or Denial of Service).
- **Reproduction Steps:** Step-by-step instructions or a minimal Proof of Concept (PoC) configuration file.
- **Scope:** Affected database drivers, command-line flags, or target configuration patterns.
- **Version Information:** The specific version or commit hash where the vulnerability was identified.

### Guidance for Database Credentials & Scrapes

When submitting details:

- **Sanitize Configurations:** Never include actual database passwords, production DSNs, or sensitive connection
  parameters in your report. Use placeholder values (e.g., `postgresql://user:pass@localhost:5432/db`).

## Dependency Vulnerabilities

Given that `sql_exporter` relies on third-party libraries and primarily drivers for database connections, it is
possible that vulnerabilities may be discovered in these dependencies. If you discover a vulnerability in a dependency,
please report it to the maintainers of that library first as they are best positioned to address the issue. If you
believe the vulnerability also affects `sql_exporter`, please report it to us privately using the methods above.

## Response Expectations

This project is maintained on a **best-effort basis** by a single developer in their spare time. While there are no
guaranteed Response Level Agreements or resolution timeframes:

- I will do my best to acknowledge reports within a reasonable timeframe.
- Confirmed critical vulnerabilities will be prioritized for patching based on available time and severity.
- Status updates will be provided via the private report thread as progress is made.

If you haven't received an acknowledgment after a few days, feel free to send a polite follow-up.

## Patch release goals

Patch release timelines will depend on the severity of the vulnerability and the complexity of the fix:

- **Critical/High:** As quickly as possible (typically within 7 days).
- **Medium/Low:** Next planned maintenance release.

Usually, releases are cut on a monthly basis, but urgent patches may be released sooner if necessary.

## Disclosure Process

When a security issue is confirmed:

1. We will work on a patch in a private fork or branch.
2. Security advisories will be drafted via GitHub Security Advisories.
3. Once the patch is merged and a release is cut, we will publish the advisory and credit the reporter (unless you
   prefer to remain anonymous).

## Security Best Practices for Users

While we strive to keep `sql_exporter` secure, deployment security relies heavily on your environment:

- **Principle of Least Privilege:** Ensure the database user utilized by `sql_exporter` has strictly read-only
  permissions limited to the required tables/system views.
- **Protect DSNs & Secrets:** Do not hardcode raw credentials in publicly accessible configuration files or version
  control. Use environment variables or secure secret managers where applicable.
- **Secure Telemetry Endpoints:** Restrict access to the HTTP metrics endpoints (`/metrics`, `/sql_exporter_metrics`)
  using network firewalls, reverse proxies (e.g., TLS/Basic Auth), or API gateways.

Thank you for helping keep `sql_exporter` safe for everyone.
