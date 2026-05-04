# AGENTS Guide — Go Fiber Clean Architecture Template

Use this file as the default operating guide for AI agents working in this repository.

## 1) Project Profile

Use this baseline profile for Go backend projects:

- Language: `Go`
- Framework/Runtime: `Fiber v2`
- Database/Storage: `PostgreSQL`
- Architecture style: `Clean Architecture`
- Main flow (mandatory): `handler -> service -> repository -> database`

If this project has additional modules, keep existing module boundaries and conventions.

## 2) Core Engineering Rules

1. Keep implementation simple, explicit, and maintainable.
2. Prefer minimal-scope changes; avoid unrelated refactors.
3. Preserve existing architecture and public behavior unless requested.
4. Do not add unnecessary dependencies.
5. Avoid exposing raw internal errors to API/UI consumers.
6. Optimize for readability first: small functions, clear naming, predictable control flow.
7. Avoid deep nesting and hidden side effects; prefer early returns and explicit branching.
8. Design with scale in mind: avoid unbounded in-memory processing for large workloads.
9. Use pagination, batching, or streaming for list/import/bulk operations.
10. Measure before/after optimization; keep only changes with clear benefit.
11. Keep performance improvements maintainable; avoid clever complexity without evidence.
12. Respect request/job cancellation and timeouts across long-running flows.
13. Preserve backward compatibility for contracts and payloads unless change is explicitly requested.

## 3) Layer Responsibilities (Strict)

### Handler
- Parse and validate input.
- Call business/service layer only.
- Return responses using the project’s response format.
- Do not access storage directly.

### Service
- Contain business logic and orchestration.
- Propagate context/cancellation/timeouts where supported.
- Coordinate data-access calls and transactions when needed.

### Repository
- Own query/persistence logic.
- Use parameterized queries/placeholders for DB operations.
- Avoid `SELECT *`; select only required fields.
- Keep query logic explicit and readable.

## 4) API/Contract Rules

- Never invent routes/endpoints/events/contracts.
- Never invent request/response fields.
- Confirm contract shape from existing router/schema/types before coding.
- Follow the existing response envelope and status conventions in this project.

## 5) Testing Rules

### General
- Inspect project structure and existing flow first.
- Validate route registration, DTO/schema behavior, middleware/interceptor behavior, and response shape.
- Ensure code compiles/builds for changed scope.

### End-to-End / Integration
- Use real app wiring and real persistence when feasible for true e2e coverage.
- Cover full chain from entry point to persistence and back.
- Use deterministic data and isolated setup/cleanup.
- Verify:
  - status code/result type
  - response schema/fields
  - side effects (when relevant)
  - key error paths

## 6) Data & Performance Guardrails

- Avoid N+1 query/access patterns.
- Prefer batching/pagination/streaming for large data.
- Use transactions for multi-step updates requiring atomicity.
- Propose indexes only when query/filter patterns justify them.
- Keep memory usage bounded for high-volume jobs.

## 7) Observability & Performance Validation

- Add structured logs for key flows without leaking sensitive data.
- Track core metrics for critical paths (latency, error rate, throughput, processed rows/items).
- Define lightweight performance checks for heavy endpoints/jobs.
- Validate performance-sensitive changes with before/after evidence when possible.

## 8) Security & Sensitive Data

- Never generate, hardcode, or expose secrets/tokens/passwords.
- Do not log sensitive values.
- Use environment variables or secret managers for credentials.

## 9) Change Safety Checklist

Before finalizing:

1. Architecture/module boundaries remain correct.
2. No shortcut bypasses core business rules.
3. Persistence/query logic stays in the proper data layer.
4. External contracts remain unchanged unless requested.
5. Build/tests for changed scope pass.