# Test-Driven Development (TDD) Rule

When implementing new features, modifying API endpoints, or bug fixing in this codebase:

1. **TDD First**: Write tests or test cases alongside code edits.
2. **Go Framework**: Use `github.com/stretchr/testify` (`assert` / `require`) for Go code.
3. **API Test Coverage**: Ensure all HTTP REST routes have unit tests using `net/http/httptest`.
4. **Verification**: Always execute `go test -v ./...` in `services/flag-service/` before completing tasks.
