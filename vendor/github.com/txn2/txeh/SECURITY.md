# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

1. **Do not** open a public issue
2. Email security concerns to cj@imti.co
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
4. Expect response within 48 hours

## Disclosure Policy

- We aim to patch critical vulnerabilities within 7 days
- Public disclosure after patch is available
- Credit will be given to reporters (unless anonymity is requested)

## Security Measures

This project follows security best practices:
- Dependencies are monitored via Dependabot
- Code is scanned with gosec and CodeQL
- Vulnerabilities are checked with govulncheck
- Supply chain security via OpenSSF Scorecard
