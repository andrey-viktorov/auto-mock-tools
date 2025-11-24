# Contributing to Auto Mock Tools

Thank you for your interest in contributing to Auto Mock Tools! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Set up the development environment:
   ```bash
   cd auto-mock-tools
   go mod download
   make build
   ```

## Development Workflow

### Making Changes

1. Create a new branch for your feature or bugfix:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. Make your changes following the code style guidelines

3. Run tests to ensure your changes don't break existing functionality:
   ```bash
   make test          # Unit tests
   make test-all      # Unit + integration tests
   ```

4. Format your code:
   ```bash
   make fmt
   ```

5. Run static analysis:
   ```bash
   go vet ./...
   ```

### Commit Messages

- Use clear and descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Reference issue numbers when applicable (e.g., "Fix #123: Description")

### Pull Requests

1. Push your changes to your fork
2. Submit a pull request to the main repository
3. Ensure all CI checks pass
4. Respond to review feedback

## Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` for code formatting
- Write clear comments for exported functions and types
- Keep functions focused and reasonably sized

## Testing

### Writing Tests

- Write unit tests for new functionality
- Place test files next to the code they test (`*_test.go`)
- Use table-driven tests where appropriate
- Aim for good test coverage

### Running Tests

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make build-testutils
make test-integration

# Run all tests
make test-all
```

## Performance Considerations

This project prioritizes performance, especially in the mock server hot path:

- Avoid allocations in request handling paths
- Use `[]byte` instead of `string` where possible
- Pre-compute and cache data at startup
- Use `sync.Pool` for frequently allocated objects
- Profile before optimizing

## Documentation

- Update README.md for user-facing changes
- Update CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/) format
- Document new CLI flags and configuration options
- Add inline code comments for complex logic

## Questions or Problems?

- Open an issue for bugs or feature requests
- Check existing issues before creating a new one
- Provide detailed information and steps to reproduce for bugs

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to Auto Mock Tools!
