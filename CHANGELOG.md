# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Auto Mock Tools
- Recording proxy (`auto-proxy`) for capturing HTTP traffic
- Mock server (`auto-mock-server`) for serving recorded responses
- Support for Server-Sent Events (SSE) recording and replay
- Scenario-based filtering with YAML configuration
- Timing replay with configurable jitter
- mTLS support for secure upstream connections
- Method-based filtering for mock responses
- Content-Type negotiation via Accept headers
- Special endpoints for stats and mock listing
- **404 request logging** - Mock server automatically logs unmatched requests to JSON files
  - Configurable log directory via `-log-dir` flag (default: `mock_log`)
  - Same file format as proxy recordings for consistency
  - Non-blocking logging that doesn't affect request handling
  - Helps identify missing mocks during development and testing
- Comprehensive test suite with integration tests
- Cross-platform build support (Linux, macOS, Windows)
- Zero-allocation hot path for mock serving
- Pre-serialized response bodies for performance

### Performance
- ~50K RPS mock serving capability
- 1 allocation per request (map lookup only)
- Pooled SSE stream writers
- Pre-computed lowercase header keys

## [0.1.0] - 2024-11-24

### Added
- Initial project setup
- Basic proxy and mock server functionality
- CI/CD with GitHub Actions
- MIT License
- Comprehensive documentation

[Unreleased]: https://github.com/andrey-viktorov/auto-mock-tools/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/andrey-viktorov/auto-mock-tools/releases/tag/v0.1.0
