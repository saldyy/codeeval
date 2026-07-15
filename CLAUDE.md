# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Internal coding-evaluation platform: problems, a Monaco-based code editor, and
auto-grading against test cases via Piston. Server-rendered Go + HTMX, no SPA
build step, no JS framework, no frontend build tooling.

## Commands

```
# Start Postgres and Piston, load schema + seed data (migrations run
# automatically via docker-entrypoint-initdb.d on first startup)
docker compose up -d

# Piston starts with no language runtimes installed - install what
# internal/piston/client.go's LanguageRuntimes needs before submissions work:
for lang in python node java go gcc; do
  curl -s -X POST http://localhost:2000/api/v2/packages \
    -H 'Content-Type: application/json' -d "{\"language\":\"$lang\",\"version\":\"*\"}"
done

# Env vars (defaults already point at the self-hosted Piston from above)
cp .env.example .env
export $(cat .env | xargs)

# Run the server
go run ./cmd/server

# Build
go build ./...

# Vet / format
go vet ./...
gofmt -l .

# Only needed when editing internal/templates/*.templ - generated
# *_templ.go files are committed, so plain `go build`/`go run` above don't
# need this:
go install github.com/a-h/templ/cmd/templ@latest
templ generate
```

There is no test suite in the repo yet. If adding one, use standard `go test ./...`.

Default login (from seed data): `test@example.com` / `password123`. Sample
problem is "Two Sum" at `/problems/two-sum`.

## Architecture

**Request flow**: `cmd/server/main.go` wires routes on an `echo.Echo` instance
(github.com/labstack/echo/v4) directly to handler methods. Handlers are
grouped by resource (`ProblemHandlers`, `SubmissionHandlers`, `AuthHandlers`
in `internal/handlers/`), each holding just a `*db.Store` (plus
`SubmissionHandlers.Piston`) — no shared template state, since templ
components are called directly (see "Templates" below). Handler methods are
`func(c echo.Context) error`.

**Auth**: `internal/handlers/auth.go` implements a deliberately minimal
scheme — a plain, unsigned `user_id` cookie (bcrypt only on login, no session
store). `RequireAuth` is an `echo.MiddlewareFunc` that reads the cookie and
stashes the user ID on the Echo context (`c.Set`/`c.Get`) for
`UserIDFromContext` to read. This is intentionally not internet-facing-safe
(see README "What's deliberately left out"); don't upgrade it into "real"
auth unless asked — it's a known, accepted tradeoff for a small trusted team
behind a VPN.

**Data layer**: `internal/db/queries.go` has hand-written SQL via `pgx/v5`
(`pgxpool.Pool`) — no ORM, no query builder, no migrations tool beyond raw SQL
files in `migrations/` applied in filename order by Postgres's
`docker-entrypoint-initdb.d`. All queries live as `Store` methods; there's no
repository-per-table split.

**Grading flow** (the core domain logic, spans several files —
`internal/handlers/submissions.go`, `internal/harness/`,
`internal/piston/client.go`, `internal/db/queries.go`). Grading is
LeetCode-style function-call submission, not stdin/stdout string
comparison: the user only ever writes/submits a function body matching the
problem's `function_signature`, never a full program.
1. `SubmissionHandlers.Create` loads the problem and *all* its test cases
   (including hidden ones where `is_sample = false`), and parses
   `problem.FunctionSignature` via `harness.ParseSignature`.
2. `harness.Wrap(sig, language, sourceCode)` concatenates the user's raw
   submission with a generated driver into one source file (per-language
   codegen in `internal/harness/{js,python,golang,java,c,cpp}.go`) that
   reads the test case's JSON-encoded call arguments from stdin, calls the
   user's function, and prints the JSON-encoded return value to stdout. A
   `submissions` row is inserted with status `pending`, storing the user's
   *raw* (unwrapped) source — the wrapped version is only ever used
   transiently for execution, never persisted.
3. For each test case, `piston.Client.RunOne` submits the wrapped source
   synchronously to Piston's `/execute` REST API (stdin = `tc.ArgsJSON`) and
   blocks for the result — there is no async job queue. This is a known
   scale limit (see README); don't silently add a queue unless asked, since
   it changes the submission lifecycle (`status` transitions, `completed_at`
   timing) that templates and `submission_results` depend on.
4. Piston has no single status field like a traditional judge — it returns
   raw exit codes/signals per stage (`compile`, `run`). `piston.Status(res)`
   synthesizes a `(ok bool, status string)` pair ("OK", "Compile Error",
   "Runtime Error (exit N)", "Killed (signal X)") from those.
5. Pass/fail is `ok` from `piston.Status` AND `reflect.DeepEqual` on the
   JSON-decoded actual stdout vs. `tc.ExpectedReturnJSON` (see
   `submissions.go` around the `passed :=` line) — comparing decoded values
   rather than raw strings makes this robust to JSON formatting/whitespace
   differences while still catching wrong values, wrong order, or wrong
   types. No floating-point tolerance.
6. Per-test-case results are persisted to `submission_results` (the status
   text column is `exec_status`, engine-neutral — it used to be
   `judge0_status` before the Piston migration), and the aggregate
   score/status is written back to the `submissions` row.
7. The handler renders `templates.SubmissionResult(...)` (no layout, just
   the results table) which HTMX swaps into the `#results` div —
   submissions never trigger a full page reload.

**Harness codegen** (`internal/harness/`): `Signature`/`Param` model a
problem's typed function name + params + return type (type vocabulary is
deliberately small: `int`, `float`, `string`, `bool`, and 1-D arrays of
each — no nested arrays/objects, no multiple return values).
`Stub(sig, language)` generates the starter code shown in Monaco;
`Wrap(sig, language, userSource)` generates the full grading driver. Every
language ends up as **one concatenated source file** sent to Piston (no
multi-file support in `piston.Client` — deliberately not needed). Three
load-bearing, non-obvious constraints discovered while building this,
worth knowing before touching any `internal/harness/*.go` file:
- **Go**: all `import` declarations must precede all other top-level
  declarations in the file (Go grammar, not just convention) — `wrapGo`
  strips the user's leading `package main` line and re-emits imports before
  re-inserting the user's function, rather than naively appending the
  driver's own imports after it.
- **Java**: Piston's Java runner treats whichever `class` is declared
  *first* in the file as the entry point, regardless of which one is
  `public` — `wrapJava` puts the generated `public class Main` before the
  user's `class Solution`, and (same import-ordering rule as Go) hoists any
  leading `import` lines the user wrote above `Main` too.
- **C**: the hand-rolled JSON tokenizer (`__splitTop`/`__toks` in the
  `cHelpers` constant) is reused for both the top-level argument split and
  each array-typed param's nested element split — `wrapC` copies the
  top-level tokens into a separate `__args` buffer *before* parsing any
  param, otherwise parsing an early array-typed param clobbers not-yet-read
  later params in the shared scratch buffer. C also follows LeetCode's own
  C convention for arrays: params get a trailing `<name>Size` int, and an
  array return type gets a trailing `int* returnSize` out-parameter,
  since a bare C pointer carries no length.
- C++ and Java's return-value serialization goes through one generic
  function (`__joinNum`/`__serialize`-style dispatch on the actual
  value/type) rather than per-signature branches; C generates the
  type-specific serialization call directly since it has neither.

**Piston client** (`internal/piston/client.go`): our language key → Piston
`{language, version}` runtime mapping lives in `LanguageRuntimes` (version
`"*"` = latest installed); extend this map (and the `<select>` in
`internal/templates/problem_detail.templ`, Monaco's `monacoLangMap`, and
`internal/harness`'s per-language `Stub`/`Wrap`) together when adding a
language. A fresh Piston instance has **no runtimes installed** — they're
installed via `POST /api/v2/packages` (see Commands above). `RunOne` clamps
`run_timeout` to `maxRunTimeoutMS` (3000) since Piston rejects the request
outright above that, regardless of what a problem's `time_limit_ms` says.

**Templates** (`internal/templates/`, [templ](https://templ.guide)): each
page is a typed Go function returning `templ.Component` — no
`map[string]any`, no string-keyed template names, no runtime template
parsing. `layout.templ` defines a `layout(title string)` component using
templ's `{ children... }` pattern (replaces the old header/footer text
blocks); page components (`Login()`, `ProblemsList(problems
[]models.Problem)`, `ProblemDetail(problem *models.Problem, samples
[]models.TestCase, stubs map[string]string)`) wrap their body with
`@layout("CodeEval") { ... }`. `SubmissionResult(...)` (the HTMX swap
target for `#results`) renders standalone, no layout — same as the old
partial. Handlers call `internal/handlers/render.go`'s `render(c, status,
component)` helper (NOT Echo's `Renderer` interface/`c.Render()` - templ
writes straight to `io.Writer` via `component.Render(ctx, w)`).
Edit `.templ` files, then run `templ generate` (see Commands) to regenerate
the matching `*_templ.go` — **the generated files are committed**, so
`go build`/`go run` never invoke the templ CLI themselves; forgetting to
regenerate after a `.templ` edit means the build silently uses stale
generated code.

`ProblemDetail` gets the per-language editor stubs (built in
`ProblemHandlers.Detail` via `harness.Stub` for each of the 6 languages) as
a plain `map[string]string` — no manual JSON marshaling in Go anymore.
`problem_detail.templ` embeds it via `@templ.JSONScript("stubs-data",
stubs)` (renders a `<script type="application/json">`), and the page's
inline Monaco-bootstrap `<script>` reads it with
`JSON.parse(document.getElementById('stubs-data').textContent)`. The
language `<select>`'s `change` handler calls `editor.setValue(stubs[lang])`
to swap the Monaco content, so add a language to `piston.LanguageRuntimes`
+ this select + `monacoLangMap` + `internal/harness` together or the editor
silently shows an empty stub for it. Monaco and HTMX are both loaded from
CDN — no bundler, no `node_modules`.

**Schema** (`migrations/001_init.sql` plus `003`-`006`, applied in filename
order): `users` → `problems` → `test_cases` → `submissions` →
`submission_results`. `test_cases.is_sample` gates whether a case is shown
on the problem page vs. grading-only (hidden). `test_cases.weight` and
`ordinal` control scoring contribution and display order respectively.
`problems.function_signature` and `test_cases.args_json`/
`expected_return_json` (all JSONB, added in `005`) are what
`internal/harness` consumes — see "Harness codegen" above.
`docker-entrypoint-initdb.d` only runs migrations on a **fresh** Postgres
volume; an existing dev volume needs new migrations applied by hand
(`docker compose exec -T postgres psql -U codeeval -d codeeval < migrations/00N_*.sql`).

## Adding problems

No admin UI — insert directly into `problems` and `test_cases` via SQL (see
README "Adding problems"). Mark grading-only cases `is_sample = false`.

## Known, intentional gaps (don't "fix" without being asked)

- No real session store (plain cookie auth)
- Synchronous, inline grading (no worker queue)
- No admin UI for authoring problems
- No rate limiting or plagiarism detection

These are documented tradeoffs in the README, not oversights.
