# Contributing to blayzen-sip

Thank you for your interest in contributing to blayzen-sip! This document provides guidelines and information for contributors.

## Code of Conduct

Please be respectful and constructive in all interactions.

## How to Contribute

### Reporting Issues

- Search existing issues before creating a new one
- Use the issue templates when available
- Include clear reproduction steps for bugs
- Provide system information (OS, Go version, Docker version)

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linters (`make test && make lint`)
5. Generate Swagger docs if API changed (`make swagger`)
6. Commit with clear messages
7. Push to your fork
8. Open a Pull Request

### Development Setup

```bash
# Clone repository
git clone https://github.com/shiv6146/blayzen-sip
cd blayzen-sip

# Install dependencies
make deps

# Start services
make docker-up

# Seed test data
make seed

# Run locally (with hot reload)
make dev
```

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `golangci-lint` to check for issues
- Add comments for exported functions
- Write tests for new features

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Run specific test
go test -v -run TestName ./...
```

### Documentation

- Update API documentation (Swagger comments) for API changes
- Update README.md for user-facing changes
- Add inline comments for complex logic

## Project Structure

```
blayzen-sip/
├── cmd/blayzen-sip/     # Main entry point
├── internal/
│   ├── api/             # REST API handlers
│   ├── call/            # Call session management
│   ├── config/          # Configuration
│   ├── models/          # Domain models
│   ├── routing/         # Call routing logic
│   ├── server/          # SIP server
│   └── store/           # Database & cache
├── migrations/          # SQL migrations
├── scripts/             # Helper scripts
├── examples/            # Example code
└── docs/                # Generated Swagger docs
```

## Questions?

Open an issue or discussion on GitHub.

