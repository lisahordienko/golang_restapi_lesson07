# LibraTrack Security Audit

Scope: production route path `cmd/api -> app.NewRouterWithDB -> Gin handlers/services/repositories`, current implementation at the time of review. The statuses record the disposition after triage.

| # | Severity | Finding | Code location | Triage | Resolution / follow-up |
|---|---|---|---|---|
| 1 | High | User passwords were not stored/verified because the original User schema had no password hash and no authentication flow. | `internal/auth/auth.go:55-70`; `internal/model/user.go:5-14`; `migrations/000003_add_auth_fields.up.sql:1-5` | Fixed Now | Added bcrypt hashing and comparison; the persisted hash is excluded from JSON and migration adds the column. |
| 2 | High | Book mutation endpoints had no authentication or role checks in the earlier HTTP router. | `internal/app/app.go:73-106`; `internal/middleware/auth.go:13-40`; `internal/middleware/rbac.go:9-29` | Fixed Now | Production routes use nested Gin groups; staff writes execute AuthMiddleware before RequireRole, and reader access receives 403. The legacy in-memory router remains only for Lesson 7 tests and is not used by `cmd/api`. |
| 3 | High | Login could reveal whether an email existed if unknown-user and wrong-password failures differed. | `internal/auth/auth.go:74-92`; `internal/handler/auth_handler.go:66-99` | Fixed Now | Both failure cases return HTTP 401 with the exact same `invalid credentials` message. Unknown accounts also perform a dummy bcrypt comparison. |
| 4 | High | User records could expose password hashes when serialized or listed by admin APIs. | `internal/model/user.go:8`; `internal/repository/user_repository.go:39-44`; `internal/app/rbac_test.go:64-84` | Fixed Now | `PasswordHash` has `json:"-"`; user-list queries select only public columns; tests assert a seeded hash is absent from JSON. |
| 5 | High | JWTs without an enforced algorithm, signature validation, or expiry could permit forged or long-lived sessions. | `internal/auth/auth.go:94-129`; `internal/middleware/auth.go:13-40` | Fixed Now | Tokens use HS256, the key is read only from `JWT_SECRET`, valid methods and `exp` are enforced at 24 hours, and missing secret fails closed. |
| 6 | Medium | Broad route bridges could unintentionally expose write methods for categories/authors/copies publicly. | `internal/app/app.go:73-100` | Fixed Now | Public Gin routes register explicit GET methods only; category/author/copy writes are in the staff group. |
| 7 | Medium | A reader could attempt to create or return loans or access another user's loan history. | `internal/app/app.go:85-100`; `internal/handler/loan_handler.go:23-72` | Fixed Now | Loan creation/return are staff-only; `/me/loans` takes user ID from validated JWT context, not request JSON. |
| 8 | Medium | SQL injection risk exists wherever request values are concatenated into raw SQL. | `internal/repository/user_repository.go:25-28`; `internal/repository/loan_repository.go:35-75` | False Positive | Reviewed auth/user/loan paths use GORM parameterized conditions and no raw request-built SQL was found. Preserve this rule for future endpoints. |
| 9 | Medium | Mass assignment could let registration/profile requests choose their own role, ID, or password hash if bound directly to a model. | `internal/handler/auth_handler.go:15-33,40-57,102-118`; `internal/handler/loan_handler.go:13-15` | Fixed Now | Handlers bind dedicated DTOs and construct/update only explicitly allowed fields. |
| 10 | Low | Password request logging could leak credentials. | `internal/middleware/request_logger.go:14-45`; `internal/app/auth_test.go:72-99` | False Positive | Request logger records metadata only and never reads/logs request bodies or headers; generic failure tests confirm response discipline. |
| 11 | Medium | Brute-force login attempts are not rate-limited. | `internal/handler/auth_handler.go:66-99` | Tracked for Later | Add an IP/account-aware shared rate limit before public deployment; avoid account lockout that enables denial of service. |
| 12 | Medium | Password policy does not screen breached passwords or offer configurable policy. | `internal/auth/auth.go:55-59` | Tracked for Later | Add policy documentation and breached-password screening; request body remains bounded and secrets are not logged. |
| 13 | Medium | Unbounded JSON bodies can consume memory during auth/profile requests. | `internal/middleware/body_limit.go:9-16`; `internal/app/app.go:63` | Fixed Now | Gin applies a 1 MiB MaxBytesReader limit before handlers bind JSON; an oversized registration integration test verifies rejection. |

## Verification

- `go test -v ./internal/app -run 'Test(Auth|RBAC)Integration'`
- `go test -race -count=1 ./...`
- `go vet ./...`
- `go build ./...`
- Migrations tested through up/down/up on an isolated PostgreSQL database.
