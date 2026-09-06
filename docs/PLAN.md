---
PLAN: "refactor!: migrate github.com/tinywasm -> webtyp.com + move form/input -> webtyp.com/input"
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — `work_schedule`: WebTyp rename + `form/input` → `input`

The framework moved `github.com/tinywasm/*` → `webtyp.com/*` (vanity path, every
module published). `origin/main` of this repo is still entirely on
`github.com/tinywasm/*`. Two jobs:

- **A.** The mechanical rename `github.com/tinywasm/*` → `webtyp.com/*`.
- **B.** The input-widget package was split out of `form` into its own module:
  `github.com/tinywasm/form/input` → **`webtyp.com/input`** (same API, same
  constructor funcs, same `input.Input` type).

Module import path **stays** `github.com/veltylabs/work_schedule`.
`github.com/veltylabs/agent_switch` on `main` just did this exact `form/input`
move — it is a working reference.

---

## A. Rename `github.com/tinywasm` → `webtyp.com`

### A1. Go source

Replace import-path prefix **`github.com/tinywasm/`** → **`webtyp.com/`** in
every `.go` file. Files on `origin/main` that reference it: `model.go`,
`model_orm.go`, `module.go`, `ops.go`, `tests/work_schedule_test.go`. Grep to
be sure: `grep -rln 'github.com/tinywasm' --include='*.go' .`. Package
selectors do not change — only the path in the `import` block.

### A2. The `form/input` split (in `model.go`)

`model.go:4` imports `"github.com/tinywasm/form/input"`. That package no longer
exists. Change it to **`"webtyp.com/input"`**. The selector stays `input.` and
every `input.Xxx(...)` call is byte-for-byte identical (`input.Checkbox()`,
`input.Text()`, `input.Number()`, …). Grep for any other occurrence:

```
grep -rn 'form/input' --include='*.go' .
```

### A3. `go.mod`

`origin/main` require block:

```
github.com/tinywasm/fmt v0.25.5
github.com/tinywasm/form v0.3.13
github.com/tinywasm/model v0.1.2
github.com/tinywasm/orm v0.11.4
github.com/tinywasm/router v0.1.19
github.com/tinywasm/storage v0.0.2-0.20260717121821-7e528006807f
github.com/tinywasm/json v0.5.17
```

- For each still-used `github.com/tinywasm/<X>`:
  `go mod edit -droprequire=github.com/tinywasm/<X>` then
  `go get webtyp.com/<X>@latest`.
- Add `go get webtyp.com/input@latest`.
- **`webtyp.com/form`**: keep it only if something still imports
  `"webtyp.com/form"` after A2 (`grep -rn '"webtyp.com/form"' --include='*.go' .`).
  If nothing does, `go mod edit -droprequire=webtyp.com/form`.
- `go mod tidy`.

`@latest` is authoritative; current tags for reference: `fmt v1.0.0`,
`form v0.4.7`, `input v0.0.6`, `model v0.1.8`, `orm v0.12.1`, `router v0.1.31`,
`storage v0.0.7`, `json v0.5.25`. No `github.com/tinywasm/*` left in `go.mod`;
no `replace … => ../…` outside this module.

### A4. Docs / config text

`*.md` / `*.yml` / `*.yaml`: `github.com/tinywasm/` → `github.com/webtyp/`.
Prose `TinyWasm`/`TinyWASM` → `WebTyp`. Leave `LICENSE` and upstream "TinyGo".

---

## Verify

```
grep -rn 'github.com/tinywasm' --include='*.go' --include='go.mod' .   # empty
grep -rn 'form/input' --include='*.go' .                               # empty
grep -rn '=> \.\./' go.mod                                            # empty
```

## Acceptance

- `go build ./...` → clean.
- `gotest ./...` → all green (vet, race, tests).
- No `github.com/tinywasm/*` anywhere in `*.go` / `go.mod`.
- `grep -rn 'form/input' --include='*.go' .` → empty.
- `go.mod` has no `webtyp.com/form` unless a `"webtyp.com/form"` import remains.

## Constraints

- **No behaviour change** — path rename + a package-path move. The `input.*`
  constructor calls and their arguments stay byte-for-byte identical apart from
  the import path.
- **No hardcoded strings** — any repeated widget-type or op literal is a named
  constant in this package.
- Keep every `//go:build` tag exactly as-is (backend + WASM shared).
