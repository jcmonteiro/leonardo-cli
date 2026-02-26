# AGENTS.md — leonardo-cli

## Project overview

Go CLI wrapping the [Leonardo.Ai REST API](https://docs.leonardo.ai/).
Two commands: `create` (start image generation) and `status` (poll by ID).
No external dependencies beyond the Go standard library.

## Build & run

```sh
go build -o leonardo ./cmd/leonardo        # compile binary
go vet ./...                                # static analysis (no linter configured)
```

## Test commands

```sh
go test ./... -short              # all unit tests, skip integration
go test ./... -v -short           # verbose
go test ./internal/service/ -v    # one package only
go test ./internal/provider/ -run TestAPIClient_CreateGeneration_SendsCorrectHTTPRequest -v  # single test
go test ./... -count=1 -short     # bypass cache
```

### Integration tests (require real API credentials)

```sh
LEONARDO_API_KEY=your-key go test ./internal/provider/ -run Integration -v
```

Integration tests are guarded by `-short` and the `LEONARDO_API_KEY` env var.
They are always skipped in normal test runs. Do not remove these guards.

## Architecture — hexagonal / clean

```
cmd/leonardo/         CLI entrypoint — flag parsing, stdout/stderr
internal/
  domain/             Value objects (GenerationRequest, GenerationResponse, GenerationStatus)
  ports/              Interface definitions (LeonardoClient) — the seam between layers
  provider/           HTTP adapter implementing LeonardoClient
  service/            Application service delegating to a LeonardoClient port
```

**Dependency rule**: domain ← ports ← service; provider implements ports.
The CLI imports domain, provider, and service but never ports directly.

## Code style

### Formatting & imports

- Format with `gofmt` (standard Go formatting, tabs, no config).
- Imports are grouped in two blocks separated by a blank line:
  1. Standard library (`fmt`, `os`, `net/http`, etc.)
  2. Internal project packages (`leonardo-cli/internal/...`)
- No third-party dependencies exist. If one is added, use a three-block grouping: stdlib, external, internal.

### Naming

- **Packages**: lowercase single words (`domain`, `ports`, `provider`, `service`).
- **Exported types**: PascalCase nouns (`APIClient`, `GenerationService`, `GenerationRequest`).
- **Constructors**: `NewXxx` pattern (`NewAPIClient`, `NewGenerationService`).
- **Interfaces**: named by capability, not `I`-prefix (`LeonardoClient`, not `ILeonardoClient`).
- **Methods**: short verb or verb-noun (`Create`, `Status`, `CreateGeneration`, `GetGenerationStatus`).
- **Receiver names**: single letter matching the type (`c` for `*APIClient`, `s` for `*GenerationService`).
- **Unexported fields**: camelCase (`apiKey`, `httpClient`).
- **CLI flags**: kebab-case (`--model-id`, `--num-images`, `--style-uuid`).
- **JSON keys**: match Leonardo API field names exactly (`num_images`, `modelId`, `styleUUID`, `guidance_scale`).

### Types

- Domain structs use plain Go types (string, int, float64, bool, []byte).
- No JSON struct tags on domain types — serialization is handled in the provider layer via `map[string]interface{}`.
- Zero-value fields are treated as "not set" and omitted from API payloads.
- Raw API responses are always preserved as `[]byte` in the `Raw` field.

### Error handling

- Wrap errors with `fmt.Errorf("context: %w", err)` — always lowercase context prefix.
- Existing context messages: `"encoding request body"`, `"creating request"`, `"executing request"`, `"reading response"`.
- Non-2xx HTTP responses: return `fmt.Errorf("API returned status %d", statusCode)` plus raw bytes in the response struct.
- Never panic. Return `(zeroValue, error)` pairs.
- In the CLI layer: print to stderr with `fmt.Fprintln(os.Stderr, ...)` then `os.Exit(1)`.

### Comments

- Every exported type and function has a `//`-style doc comment starting with the name.
- Comments use two spaces after periods within the same sentence cluster (Go convention).
- File-level or section-level comments explain *why*, not *what*.

### Struct construction

- Use `NewXxx` constructors, not bare struct literals outside the defining package.
- Constructors should apply defaults (e.g., `NewAPIClient` creates a 60s-timeout `http.Client` when nil).

## Testing conventions

### Test behaviors, not implementations

- Tests target behaviors at port boundaries, not internal class structure.
- Service tests use a **fake implementation** of `ports.LeonardoClient` with injectable function fields.
- Provider/adapter tests use `net/http/httptest.Server` with a `rewriteTransport` to redirect requests to a local server.
- Never mock domain types or internal collaborators. Only mock other ports.

### Test file placement

- Test files use `_test` package suffix (`package service_test`, `package provider_test`) for black-box testing.
- Integration test files are named `integration_test.go`, placed alongside the code they test.

### Test naming

- Service tests: `TestCreate_*`, `TestStatus_*` — verb describing the behavior.
- Provider tests: `TestAPIClient_CreateGeneration_*`, `TestAPIClient_GetGenerationStatus_*`.
- Integration tests: `TestIntegration_*` prefix.
- Name describes the scenario, not the implementation: `ReturnsGenerationIDAndRawResponse`, not `CallsCreateGeneration`.

### Assertions

- Use stdlib `testing` only — no assertion libraries.
- `t.Fatalf` for conditions that make continuing pointless (nil checks before dereferencing).
- `t.Errorf` for everything else so multiple failures are reported.
- Compare expected vs actual: `"expected %q, got %q"` pattern.

### Integration test guards

Integration tests must check **both** conditions before running:

```go
if testing.Short() {
    t.Skip("skipping integration test in short mode")
}
if os.Getenv("LEONARDO_API_KEY") == "" {
    t.Skip("skipping integration test: LEONARDO_API_KEY not set")
}
```

## Compile-time interface checks

Ensure adapters satisfy their ports with a blank identifier assignment:

```go
var _ ports.LeonardoClient = (*APIClient)(nil)
```

## Git conventions

- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.
- Commit granularly — one logical change per commit.
- Never commit secrets, `.env` files, or API keys.
- `LEONARDO_API_KEY` is always read from the environment at runtime.
