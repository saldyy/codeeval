# CodeEval

Internal coding-evaluation platform: problems, Monaco-based code editor, and
auto-grading against test cases via Piston. Grading is LeetCode-style
function-call submission (the editor shows a typed function stub per
problem/language; the platform calls it directly with structured
arguments and checks the structured return value) rather than raw
stdin/stdout. Server-rendered Go + HTMX, no SPA build step.

## Stack

- **Go** (stdlib `net/http` routing via Echo — see `cmd/server/main.go`)
- **[templ](https://templ.guide)** for HTML rendering — typed Go components
  compiled from `.templ` source (`internal/templates/`), not runtime-parsed
  `html/template`
- **HTMX** for the submit-and-see-results interaction (loaded from CDN)
- **Monaco editor** for the code input (loaded from CDN)
- **PostgreSQL** for problems, test cases, submissions
- **Piston** (self-hosted by default) for sandboxed code execution

## Local setup

1. Start Postgres and Piston, and load the schema + seed data:
   ```
   docker compose up -d
   ```
   The `migrations/*.sql` files are mounted into `docker-entrypoint-initdb.d`
   and run automatically on first startup. Piston starts with **no language
   runtimes installed** — you must install them before submissions will work.

2. Install the runtimes this app uses (maps to the languages in
   `internal/piston/client.go`'s `LanguageRuntimes` — `gcc` provides both `c`
   and `c++`, `node` provides `javascript`):
   ```
   for lang in python node java go gcc; do
     curl -s -X POST http://localhost:2000/api/v2/packages \
       -H 'Content-Type: application/json' \
       -d "{\"language\":\"$lang\",\"version\":\"*\"}"
   done
   ```
   Check `curl http://localhost:2000/api/v2/packages` to confirm they show
   `"installed": true`.

3. Copy the env file (defaults already point at the self-hosted Piston
   instance from step 1; `PISTON_API_KEY` is only needed if you point
   `PISTON_URL` at a gated/hosted instance instead):
   ```
   cp .env.example .env
   ```

4. Run the server:
   ```
   export $(cat .env | xargs)
   go run ./cmd/server
   ```

5. Visit `http://localhost:8080`, log in with `test@example.com` /
   `password123` (from the seed data), and open the "Two Sum" problem.

Generated `*_templ.go` files (compiled from `internal/templates/*.templ`)
are committed, so the steps above don't need the templ CLI. Only install it
if you're editing a `.templ` file:
```
go install github.com/a-h/templ/cmd/templ@latest
templ generate   # regenerate after any .templ edit, before building/committing
```

## Adding problems

Insert rows into `problems` and `test_cases` directly for now (or write a
small admin CLI/HTTP form later — not included in this skeleton). Mark
grading-only cases with `is_sample = false` so they stay hidden from the UI
but still count toward the score.

Each problem needs a `function_signature` (JSONB) describing its typed
function name, params, and return type — this drives both the per-language
starter stub shown in the editor and the generated grading driver (see
`internal/harness`). Supported types: `int`, `float`, `string`, `bool`, and
one-dimensional arrays of each (`int[]`, `float[]`, etc). Each test case
needs `args_json` (call arguments, in order) and `expected_return_json`
(the expected return value), both JSONB. Example, for a `maxProduct(nums:
int[]) -> int` problem:

```sql
INSERT INTO problems (slug, title, description_md, difficulty, time_limit_ms, memory_limit_kb, function_signature)
VALUES ('max-product', 'Maximum Product Subarray', '...', 'medium', 3000, 256000,
  '{"function_name": "maxProduct", "params": [{"name": "nums", "type": "int[]"}], "return_type": "int"}');

INSERT INTO test_cases (problem_id, args_json, expected_return_json, is_sample, weight, ordinal)
SELECT id, '[[2,3,-2,4]]', '6', true, 1, 0 FROM problems WHERE slug = 'max-product';
```

## Running Piston elsewhere

`docker-compose.yml`'s `piston` service runs in `privileged: true` mode,
which Piston requires for its isolate/cgroups-based sandboxing — keep that in
mind if you move this to a k8s cluster later (privileged pods need explicit
allowance). Point `PISTON_URL` at any other Piston deployment (in-cluster,
or a gated hosted instance with `PISTON_API_KEY` set) using the same REST API.

Piston itself caps `run_timeout` at 3000ms regardless of what a problem's
`time_limit_ms` says (`internal/piston/client.go` clamps to this), and the
JVM needs a noticeably higher `memory_limit_kb` than the other languages
just to start up (~150-160MB baseline) — the seed problems use
`memory_limit_kb = 256000` accordingly.

## What's deliberately left out of this skeleton

- **Real session store**: auth is a plain user-ID cookie, fine for a small
  trusted team behind a VPN/internal network, not fine for anything
  internet-facing. Swap for signed sessions or SSO before exposing this
  externally.
- **Async grading queue**: submissions run synchronously against Piston
  inline in the request. Fine at low concurrency; move to a worker queue
  (e.g. a Postgres-backed job table or NATS) if submission volume grows.
- **Admin UI** for authoring problems — currently SQL-only.
- **Rate limiting / plagiarism detection** — out of scope for v1 per your
  earlier answer, but the `submissions` table already has what you'd need
  (source code + timestamps) to add similarity checks later.

## Project layout

```
cmd/server/          entrypoint, route wiring
internal/db/         connection pool + hand-written queries
internal/piston/     Piston API client
internal/harness/    per-language function stub + grading driver codegen
internal/handlers/   HTTP handlers (problems, submissions, auth)
internal/models/     domain structs
internal/templates/  templ components (.templ source + generated *_templ.go)
migrations/          schema + seed SQL
static/               CSS
```
