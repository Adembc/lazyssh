# Important Commands for Lazyssh Development

## Development Workflow

### Basic Operations
- `git status` - Check current repository state
- `git add .` - Stage all changes
- `git commit -m "message"` - Commit changes
- `git push origin main` - Push to remote
- `git pull origin main` - Pull latest changes

### Building & Running
- `make build` - Build the binary with quality checks
- `make run` - Run the application from source code
- `./bin/lazyssh` - Run the built binary
- `make build-all` - Build binaries for all platforms (Linux/Mac/Windows)

### Code Quality & Testing
- `make quality` - Run all quality checks (fmt, vet, lint)
- `make fmt` - Format code with gofumpt and go fmt
- `make lint` - Run golangci-lint
- `make lint-fix` - Run golangci-lint with automatic fixes
- `make test` - Run unit tests with race detection and coverage
- `make coverage` - Generate and open coverage report
- `make benchmark` - Run benchmark tests

### Dependency Management
- `go mod download` - Download dependencies
- `go mod verify` - Verify dependencies
- `go mod tidy` - Clean up dependencies
- `make deps` - Download and verify dependencies
- `make update-deps` - Update all dependencies to latest

### Maintenance
- `make clean` - Clean build artifacts and caches
- `make version` - Display version information
- `make help` - Show all available make targets

### Quick Commands
- `make run-race` - Run with race detector enabled
- `make test-verbose` - Run tests with verbose output
- `make check` - Run staticcheck analyzer

### System Prerequisites (macOS)
- `brew install go` - Install Go
- `go install golang.org/x/tools/gopls@latest` - Install language server
- Standard unix tools: `ls`, `cd`, `grep`, `find`, `mkdir`, `cp`, `mv`

## Semantic PRs
Use semantic PR titles: `feat(scope): description`, `fix(scope): description`, `refactor(scope): description`
Allowed scopes: ui, cli, config, parser