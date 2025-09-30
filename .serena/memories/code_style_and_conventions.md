# Code Style and Conventions for Lazyssh

## Go Code Style
- **Formatting**: Use `gofumpt` (advanced gofmt) for formatting
- **Imports**: Standard library first, then third-party packages, then local packages
- **Naming**: Use Go naming conventions (PascalCase for exported, camelCase for unexported)
- **Comments**: Use complete sentences starting with name of declared entity
- **Error Handling**: Return errors explicitly, use error wrapping
- **Context**: Pass context.Context for cancellable operations

## File Organization
- **Entry Point**: `cmd/main.go` with cobra root command
- **Clean Architecture**:
  - `internal/core/` - domain entities and services
  - `internal/adapters/` - UI and data adapters
  - Repository pattern for data persistence
- **Configuration**: Use dependency injection pattern

## Documentation
- **README**: Comprehensive with screenshots and installation instructions
- **License**: Apache License 2.0
- **Headers**: All Go files require copyright header

## Pull Request Conventions
- **Semantic PRs**: Use conventional commits format
  - `feat:` new features
  - `fix:` bug fixes
  - `improve:` UX improvements
  - `refactor:` code changes
  - `docs:` documentation
  - `test:` adding tests
  - `ci:` CI/CD changes
  - `chore:` maintenance
  - `revert:` reverts
- **Scopes**: ui, cli, config, parser (optional)

## Testing
- **Unit Tests**: Race detection (`-race` flag)
- **Coverage**: Generate HTML coverage reports
- **Benchmarks**: Use standard Go benchmark format
- **Test Files**: Place alongside source `_test.go`

## Linting & Quality
- **golangci-lint**: Comprehensive linter suite
- **Enabled Linters**: ~25 linters including staticcheck, revive, gocritic
- **Disabled Rules**: Some relaxations for internal packages and tests
- **Copyright Headers**: Enforced via goheader linter
- **Spelling**: US English locale

## Architecture Patterns
- **Hexagonal Architecture** (Ports & Adapters)
- **Dependency Injection** for testability
- **Repository Pattern** for data access
- **Service Layer** for business logic
- **Logging**: Structured logging with zap

## SSH Integration
- **Security First**: Non-destructive config writes with backups
- **Atomic Writes**: Temporary files then rename to prevent corruption
- **Multiple Backups**: Original backup + rotate 10 timestamped backups
- **Permissions**: Preserve file permissions on SSH config

## Internationalization
- **Language**: English US (misspell locale: US)
- **Error Messages**: Clear and actionable
- **UI Text**: Consistent terminology