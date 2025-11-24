# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within Auto Mock Tools, please send an email to the maintainers. All security vulnerabilities will be promptly addressed.

**Please do not report security vulnerabilities through public GitHub issues.**

### What to Include

When reporting a vulnerability, please include:

1. Type of vulnerability (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
2. Full paths of source file(s) related to the manifestation of the vulnerability
3. Location of the affected source code (tag/branch/commit or direct URL)
4. Step-by-step instructions to reproduce the issue
5. Proof-of-concept or exploit code (if possible)
6. Impact of the vulnerability, including how an attacker might exploit it

### Response Timeline

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will provide a detailed response within 7 days, including our evaluation and expected resolution timeline
- We will notify you when the vulnerability is fixed

### Disclosure Policy

- Security issues will be disclosed publicly after a fix has been released
- We will credit researchers who report valid security issues (unless they prefer to remain anonymous)

## Security Best Practices

When using Auto Mock Tools:

1. **Secure Mock Storage**: Mock files may contain sensitive data (API keys, tokens, etc.)
   - Store mock files in a secure location
   - Add mock directories to `.gitignore` if they contain sensitive data
   - Use file system permissions to restrict access

2. **Network Security**:
   - The proxy and mock server bind to `127.0.0.1` by default
   - Use firewall rules when binding to `0.0.0.0`
   - Consider using TLS/mTLS for production scenarios

3. **x-mock-id Header**:
   - This header is stripped from upstream requests and client responses
   - Never expose it in production API responses

4. **Input Validation**:
   - Validate and sanitize mock files if accepting them from external sources
   - Be cautious with mock files from untrusted sources

## Known Limitations

1. **TLS Certificate Validation**: The proxy uses `InsecureSkipVerify: true` by default for development/testing. In production, configure proper certificate validation.

2. **Resource Limits**: The mock server loads all mocks into memory. Large mock collections may consume significant memory.

3. **DOS Protection**: No built-in rate limiting. Use a reverse proxy with rate limiting for production deployments.

## Updates and Patches

Security updates will be released as soon as possible after a vulnerability is confirmed. Update to the latest version to ensure you have all security patches.

Subscribe to release notifications on GitHub to stay informed about security updates.
