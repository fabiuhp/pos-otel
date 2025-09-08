# Repository Guidelines

## Project Structure & Module Organization
- Root: `go.mod` at the repository root.
- Binaries: `cmd/<app>/main.go` (one folder per executable).
- Libraries: `internal/<pkg>` (private) and `pkg/<pkg>` (public APIs when needed).
- APIs and schemas: `api/` (OpenAPI/Protos), configs in `configs/`.
- Tests: co-located as `*_test.go` next to implementation.
- Tooling & CI: `scripts/` for helpers, workflows in `.github/`.

## Build, Test, and Development Commands
- Build binary: `go build ./cmd/<app>`
- Run locally: `go run ./cmd/<app>`
- Unit tests: `go test ./...`
- Race + coverage: `go test -race -coverprofile=coverage.out ./...` then `go tool cover -func=coverage.out`
- Lint/Vet: `golangci-lint run` and `go vet ./...`
- Format: `gofmt -s -w .` or `go fmt ./...`
If present, prefer `make build`, `make test`, `make lint`, `make run` wrappers.

## Coding Style & Naming Conventions
- Use `gofmt`/`goimports`; do not hand-format. CI should verify formatting.
- Package names: short, lowercase, no underscores or plurals (avoid stutter).
- Exported identifiers: PascalCase; unexported: camelCase. Constants in PascalCase; errors as `var ErrX = errors.New("...")`.
- Error handling: wrap with `fmt.Errorf("context: %w", err)`; use `errors.Is/As`.
- Context-first: accept `ctx context.Context` as the first parameter at boundaries.

## Testing Guidelines
- Prefer table-driven tests and `t.Parallel()` for independent cases.
- Keep tests close to code: `foo_test.go` for `foo.go`.
- Coverage goal: ≥80% for critical packages; include error-path tests.
- Use temporary dirs via `t.TempDir()` and avoid external network calls in unit tests.

## Commit & Pull Request Guidelines
- Commits: Conventional Commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`). Imperative, scoped messages.
- PRs: clear summary, linked issues (`Closes #123`), test evidence, and notes for config/migrations. Keep diffs focused and small.

## Security & Configuration Tips
- Never commit secrets. Provide `.env.example`; load via env at runtime.
- Set timeouts on `http.Client`/servers; validate inputs at boundaries.
- If observability is used, centralize OpenTelemetry in `internal/telemetry/`.
