# What to do when a development task is completed

## Pre-Commit Quality Checks (Mandatory)
Always run these before committing changes:

```bash
# Run all quality checks
make quality

# This executes:
# - make fmt (formatting with gofumpt + gofmt)
# - make vet (static analysis)
# - make lint (golangci-lint with all configured linters)
```

## Testing (Mandatory)
```bash
# Run tests with race detection and coverage
make test

# Generate coverage report if needed
make coverage
```

## Build Verification
```bash
# Ensure the code builds successfully
make build

# Test that the application runs
make run
```

## Linting & Static Analysis
```bash
# Fix any auto-fixable linting issues
make lint-fix

# Run staticcheck analyzer
make check
```

## Dependency Management
```bash
# Keep dependencies clean
go mod tidy
```

## Commit Message Format
Use semantic commit messages:
- `feat(ui): add server ping functionality`
- `fix(config): handle empty SSH config files`
- `improve(performance): optimize server list rendering`
- `refactor(core): extract server parsing logic`
- `test(services): add unit tests for server validation`
- `docs: update installation instructions`
- `ci: add automated testing workflow`
- `chore: update copyright headers`

## Code Review Checklist
- [ ] All quality checks pass (`make quality`)
- [ ] Tests pass with coverage maintained
- [ ] No new linting violations
- [ ] Code follows established patterns and conventions
- [ ] Security implications considered (especially SSH handling)
- [ ] Performance impact assessed
- [ ] Documentation updated if needed
- [ ] Changelog updated for user-facing changes

## SSH Configuration Safety
When modifying SSH functionality:
- [ ] Backups are properly created
- [ ] Atomic writes are used (temp file then rename)
- [ ] File permissions are preserved
- [ ] Error handling includes rollback scenarios
- [ ] No exposure of sensitive data (private keys, passwords)

## Final Verification
```bash
# One final build and test
make clean && make build
./bin/lazyssh --help  # Verify binary works
```

## Staging and Push
```bash
# Stage and commit
git add .
make quality  # Final check
make test     # Final test
git commit -m "feat(description): your semantic message"
git push origin main
```