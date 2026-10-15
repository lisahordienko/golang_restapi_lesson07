## Capstone Project: LibraTrack (Library Management System)

**Time Budget:** Up to 2 hours.  
**Objective:** Secure the **LibraTrack** REST API following production-grade security standards. Implement full authentication using password hashing (`bcrypt`) and token issuance (`JWT`), develop role-based authorization using middleware (`RBAC Middleware`), and conduct a structured AI-assisted security review of your codebase to identify and remediate vulnerabilities.

---

### PART 1: Manual Tasks (Security Architecture, Modeling & Review) [~35 mins]
*This section covers tasks requiring critical human design decisions, risk analysis, and code quality gates.*

* **Task 1.1: Role, Access Matrix, and DTO Design (Security & Contract Design):**
  1. Create a `security_design.txt` file (or add a section to your `README.md`).
  2. Define the system roles: a minimum of 2 roles (`reader` and `admin`) as required by the assignment (or 3 roles: `reader`, `librarian`, `admin` for the full capstone model).
  3. Map out the access control matrix for LibraTrack endpoints:
     * **Public (Unauthenticated):** `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `GET /api/v1/books`.
     * **Protected (Any Authenticated User):** `GET /api/v1/me/loans`.
     * **Staff (`librarian` / `admin`):** `POST /api/v1/books`, `PUT /api/v1/books/:id`, `POST /api/v1/loans`, `POST /api/v1/loans/:id/return`.
     * **Admin (`admin`):** `GET /api/v1/users`, `PUT /api/v1/users/:id/role`.
  4. Design request DTOs to prevent Mass Assignment vulnerabilities: ensure `RegisterRequest` and `UpdateProfileRequest` contain only user-editable fields, and attach the `json:"-"` struct tag to `PasswordHash` to prevent exposing hashes in JSON responses.

* **Task 1.2: Manual Security Audit of Password Handling & Logging:**
  1. Audit the `/auth/login` handler: verify that authentication failures (both unknown email and invalid password) return an identical HTTP 401 error message (`"invalid credentials"`) to prevent User Enumeration attacks.
  2. Inspect the `LoggingMiddleware` from Lesson 7: ensure that request-logging middleware redacts or skips the `password` field in request bodies.
  3. Verify middleware abort behavior: ensure that authentication/authorization failures explicitly invoke `c.AbortWithStatusJSON()` to halt the Gin handler execution chain.

---

### PART 2: Prompt Engineering (Authentication, JWT & RBAC Middleware) [~40 mins]
*This section focuses on writing precise, highly-specified prompts for your AI assistant to generate authentication and authorization code.*

* **Task 2.1: Prompting the Authentication Module (bcrypt + JWT issuance):**
  * Construct and execute a prompt to generate DB migration files (`000003_add_auth_fields.up.sql` / `.down.sql` adding `password_hash` and `role` to `users`) and update the Go `User` model.
  * *Prompt Requirements for `Register` and `Login` Handlers:*
    1. Registration must hash passwords using `bcrypt.GenerateFromPassword` with `bcrypt.DefaultCost` (10) and default the role to `reader`.
    2. Login must verify passwords using `bcrypt.CompareHashAndPassword` and issue a signed JWT using `jwt.SigningMethodHS256`.
    3. The JWT Payload must carry claims: `user_id`, `role`, and expiration timestamp `exp` (24 hours). The signing secret must be loaded strictly from the environment variable `os.Getenv("JWT_SECRET")`.
  * *Deliverable:* Save your prompt text to `prompts.txt`, and integrate generated files into `internal/auth/` and `internal/handler/`.

* **Task 2.2: Prompting Middleware Chain (`AuthMiddleware` & `RequireRole`):**
  * Write a prompt to generate `AuthMiddleware` (`internal/middleware/auth.go`) and `RequireRole` (`internal/middleware/rbac.go`).
  * *Prompt Requirements:*
    1. `AuthMiddleware` must extract tokens from the `Authorization: Bearer <token>` header, validate the signature and `exp` claim, and store claims in the Gin context via `c.Set("user_id", ...)` and `c.Set("role", ...)`.
    2. `RequireRole` must be variadic (`allowed ...string`), inspect `c.Get("role")`, and return HTTP 401 Unauthenticated (if unauthenticated) or HTTP 403 Forbidden (if role is not authorized).
    3. The prompt must demonstrate route registration in `main.go` using nested Gin route groups (`public`, `protected`, `staff`, `admin`) with correct middleware execution order (Auth first, then RBAC).
  * *Deliverable:* Save your prompt to `prompts.txt`.

---

### PART 3: Agentic AI Use (Automated AI Security Audit & Testing) [~45 mins]
*Leverage your AI assistant as an autonomous partner to perform a security audit and generate comprehensive tests.*

* **Task 3.1: AI-Assisted Security Code Review:**
  * Supply your AI agent with the code of your created modules (`auth`, `middleware`, `book_handler`, `loan_handler`).
  * *Agent Prompt Instruction:*  
    > "Review this Go/Gin API code for security vulnerabilities. Check specifically for: 1) SQL injection (raw queries, string concatenation); 2) Mass assignment (binding request JSON directly to DB models); 3) Missing authentication or RBAC checks; 4) Sensitive data exposure (leaking password hashes in JSON responses). Output a structured list with line numbers, severity (Low/Medium/High), and proposed fixes. Do not rewrite files completely."
  * *Triage & Report:* Critically evaluate the AI output. Compile a report in `security_audit.md` documenting **at least 5 findings** with triage statuses: *Fixed Now*, *Tracked for Later*, or *False Positive*.

* **Task 3.2: AI-Generated Table-Driven Security Tests:**
  * Instruct your AI agent to write table-driven integration tests in `auth_test.go` and `rbac_test.go`.
  * *Agent Instruction:*  
    > "Write table-driven integration tests for auth and RBAC using httptest. Cover test cases: 1) Registering duplicate email (HTTP 422); 2) Valid login returning JWT (HTTP 200); 3) Login with wrong password (HTTP 401 generic error); 4) Calling protected route with missing/expired token (HTTP 401); 5) Reader calling POST /api/v1/books (HTTP 403 Forbidden); 6) Admin/Librarian calling POST /api/v1/books (HTTP 201 Created)."
  * *Execution:* Run tests locally using `go test -v ./...` and ensure all test cases pass.

---

### Evaluation Criteria & Deliverables (Self-Check)

Submit a link to your repository containing:
1. **DB Migrations:** `000003_add_auth_fields.up.sql` and `.down.sql`.
2. **Secure Password Handling:** Passwords hashed with `bcrypt` on registration, and `PasswordHash` annotated with `json:"-"`.
3. **Error & Token Discipline:** User enumeration prevented on `/login`, JWT includes `exp` and signs using `os.Getenv("JWT_SECRET")`.
4. **Middleware Ordering:** `main.go` cleanly isolates route groups (`public`, `protected`, `staff`, `admin`), executing `RequireRole` after `AuthMiddleware`.
5. **Prompt Log `prompts.txt`:** Contains prompt definitions for Tasks 2.1 and 2.2.
6. **Security Audit `security_audit.md`:** Documents at least 5 identified vulnerabilities with severity ratings and triage fixes.
7. **Test Coverage:** Integration tests verify `401 Unauthenticated` and `403 Forbidden` behaviors across `reader`, `librarian`, and `admin` roles.

---
