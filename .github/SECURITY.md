# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest | :x:                |

## Reporting a Vulnerability

Please report (suspected) security vulnerabilities to **[security@kanzi.dev](mailto:security@kanzi.dev)**. You will receive a response within 48 hours. If the issue is confirmed, we will release a patch as soon as possible depending on complexity but historically within a few days.

### Important Notes

**kindplane is designed for local development and testing only.** It is not intended for production use. Security vulnerabilities should be reported if they:

- Could lead to unauthorised access to local development environments
- Could compromise the local machine running kindplane
- Could expose sensitive credentials or configuration data
- Could allow arbitrary code execution on the host system

### What to Include

When reporting a security vulnerability, please include:

- A description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if you have one)
- Your contact information (optional, but helpful for follow-up questions)

## Disclosure Policy

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will confirm the vulnerability and determine affected versions within 7 days
- We will keep you informed of our progress
- We will notify you when the vulnerability has been fixed
- We will credit you in the security advisory (unless you prefer to remain anonymous)

## Security Best Practices

When using kindplane:

- Never commit `kindplane.yaml` files containing real credentials to version control
- Use `.gitignore` to exclude local configuration files
- Regularly update kindplane to the latest version
- Review and understand the Kubernetes resources created by kindplane
- Use kindplane only in isolated development environments

## Scope

This security policy applies to:

- The kindplane CLI tool
- Official documentation
- GitHub Actions workflows in this repository

This security policy does not apply to:

- Third-party dependencies (report to their respective maintainers)
- Kubernetes, Kind, Crossplane, or other tools managed by kindplane (report to their respective projects)
- Issues in local Kind clusters created by kindplane (these are isolated development environments)
