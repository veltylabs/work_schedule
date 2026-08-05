---
PLAN: "feat: work_schedule joins the reusable-module harness (OpModule, storage/mem tests, drop mcp fork)"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 9885113582111974685
---

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

# Plan — `work_schedule` joins the reusable-module harness

You have **zero prior context** and **only this repository** (`github.com/veltylabs/work_schedule`).
This plan is self-contained: every contract, version, and code change you need is inline or in this
repo's own `AGENTS.md` (read that file in full before starting — it is the canonical
whitelist/blacklist/testing contract every `veltylabs/modules/*` repo holds to; this plan does not
repeat its theory, only the concrete changes for this module).

Reference implementation (the pattern this plan replicates): `github.com/veltylabs/item_catalog` —
its `.go` files show the target shape. This module is much smaller (one read-only op, no writes, no
schema it owns), so several of item_catalog's parts (Deps, `ddl`, `events`, `view`) are **deliberately
absent here** — do not add them speculatively. Each stage below says explicitly whether the ecosystem
default applies or is exempted for this module, and why.

## 0. What this plan fixes — three unrelated things, one dispatch

1. **Anti-pattern bug**: `go.mod` has `replace github.com/tinywasm/mcp => ../../../tinywasm/mcp` — a
   local fork/vendor of an upstream repo. `AGENTS.md`'s blacklist forbids this unconditionally ("A
   `replace` pointing at a local path is always a defect to close"). Fixed by Stage 3/7: once
   `tinywasm/mcp` is dropped as a dependency entirely, the `replace` line has nothing to point at and
   is deleted with it.
2. **Doc consolidation**: this repo currently carries THREE generations of migration docs —
   `docs/PLAN.md` (mid-quality, targets stale `model@v0.0.14`/`orm@v0.9.28`), `docs/CHECK_PLAN.md`
   (oldest, targets `orm@v0.6.0`/`fmt@v0.22.2`, fully superseded), and `docs/SKILL.md` (still
   accurate in spirit, stale in file/method names). This plan **is** the merged, current replacement
   for the first two. `docs/CHECK_PLAN.md` must be **deleted** (Stage 8) — nothing in it is missing
   from this plan; its only content not already covered here was the `req.Bind(&args)` binding
   mechanism, which disappears anyway once `mcp.Request` is replaced by `router.Context.Decode` (see
   Stage 3). `docs/SKILL.md` is **updated in place**, not deleted (Stage 9).
3. **The harness migration itself**: drop `mcp.ToolProvider`/`unixid`/`sqlite`, adopt
   `router.OpModule`/`ddl`-free persistence/`storage/mem` tests — detailed stage by stage below.

## 1. Current state — the exact files as they exist today

### `go.mod` (current)

```
module github.com/veltylabs/work_schedule

go 1.25.2

require (
	github.com/tinywasm/context v0.0.18
	github.com/tinywasm/fmt v0.24.0
	github.com/tinywasm/form v0.2.6
	github.com/tinywasm/json v0.5.2
	github.com/tinywasm/mcp v0.1.1
	github.com/tinywasm/orm v0.6.0
	github.com/tinywasm/sqlite v0.2.0
)

require (
	... (indirect deps, including github.com/tinywasm/unixid v0.2.23 // indirect)
)

replace github.com/tinywasm/mcp => ../../../tinywasm/mcp
```

### `model.go` (current — struct + tags, to be replaced)

```go
//go:build !wasm

package workschedule

// Staff maps to the legacy 'staff' table. READ-ONLY — no DDL allowed.
type Staff struct {
	ID           int64  `db:"pk"`
	Name         string `db:"not_null"`
	Role         string `db:"not_null"`
	Email        string `db:"unique"`
	PasswordHash string `db:"-"` // ormc:exclude
}

func (s *Staff) TableName() string { return "staff" }

// WorkCalendar maps to the legacy 'workcalendar' table. READ-ONLY — no DDL allowed.
// ormc:model workcalendar
// ormc:table workcalendar
type WorkCalendar struct {
	ID        int64  `db:"pk"`
	StaffID   int64  `db:"ref=staff,not_null"`
	DayOfWeek int    `db:"not_null"` // 0=Sunday … 6=Saturday
	StartTime string `db:"not_null"` // "HH:MM"
	EndTime   string `db:"not_null"` // "HH:MM"
	IsActive  bool   `db:"not_null"`
}

func (w *WorkCalendar) TableName() string { return "workcalendar" }

// ormc:formonly
type getWorkScheduleArgs struct {
	StaffID int64 ``
}

// ormc:formonly
type scheduleEntry struct {
	Day      int    ``
	DayName  string ``
	IsActive bool   ``
	Start    string `json:",omitempty"`
	End      string `json:",omitempty"`
}

// ormc:formonly
type staffResponse struct {
	StaffName string          ``
	StaffRole string          ``
	Schedule  []scheduleEntry ``
}
```

### `mcp.go` (current — to be split into `module.go` + `ops.go`)

```go
//go:build !wasm

package workschedule

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/orm"
)

type Module struct {
	db *orm.DB
}

func New(db *orm.DB) *Module { return &Module{db: db} }

func (m *Module) RegisterTools(srv *mcp.Server) {
	for _, t := range m.Tools() {
		srv.AddTool(t)
	}
}

func (m *Module) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "get_work_schedule",
			Description: "Returns the work calendar for a staff member.",
			Action:      'r',
			Execute:     m.GetWorkSchedule,
		},
	}
}

func (m *Module) GetWorkSchedule(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args getWorkScheduleArgs
	if err := req.Bind(&args); err != nil {
		return &mcp.Result{IsError: true, Content: fmt.Err("params", "invalid").Error()}, nil
	}
	staffID := args.StaffID

	staffModel := &Staff{}
	staffRow, err := ReadOneStaff(
		m.db.Query(staffModel).Where(Staff_.ID).Eq(staffID),
		staffModel,
	)
	if err != nil || staffRow == nil {
		return &mcp.Result{IsError: true, Content: fmt.Err("staff", "not", "found").Error()}, nil
	}

	calRows, err := ReadAllWorkCalendar(
		m.db.Query(&WorkCalendar{}).
			Where(WorkCalendar_.StaffID).Eq(staffID).
			OrderBy(WorkCalendar_.DayOfWeek).Asc(),
	)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}

	resp := buildStaffResponse(staffRow, calRows)
	var s string
	if err := json.Encode(&resp, &s); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(s), nil
}

var dayNames = [7]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

func buildStaffResponse(s *Staff, rows []*WorkCalendar) staffResponse {
	entries := make([]scheduleEntry, len(rows))
	for i, r := range rows {
		dayIndex := r.DayOfWeek
		if dayIndex < 0 || dayIndex > 6 {
			dayIndex = 0
		}
		e := scheduleEntry{Day: r.DayOfWeek, DayName: dayNames[dayIndex], IsActive: r.IsActive}
		if r.IsActive {
			e.Start = r.StartTime
			e.End = r.EndTime
		}
		entries[i] = e
	}
	return staffResponse{StaffName: s.Name, StaffRole: s.Role, Schedule: entries}
}
```

`mcp_test.go` (current) builds `*orm.DB` via `sqlite.Open(":memory:")`, calls `db.CreateTable(&Staff{})`
/`db.CreateTable(&WorkCalendar{})` directly on the sqlite handle (legal there because it is the test's
own scratch DB, not the legacy one — but still a forbidden concrete-driver import per `AGENTS.md`),
and drives `m.GetWorkSchedule` through `mcp.Request{Params: mcp.CallToolParams{...}}`.

**Read-only property already holds today**: nothing in `mcp.go` ever calls `db.Create`/a
migration for `Staff`/`WorkCalendar` in **production** code (`New()` only stores the `*orm.DB`) — only
the *test* seeds via a scratch sqlite DB it owns. Preserve this property exactly; see Stage 5.

## 2. Target file layout

```
work_schedule/
├── model.go          # model.Definition literals — NO build tag (isomorphic, see AGENTS.md)
├── module.go         # Module struct, New(db), GetWorkSchedule() — NO build tag
├── ops.go            # ModelName() + MountOps(router.OpRegistry) — NO build tag
├── model_orm.go      # regenerated by ormc — DO NOT hand-edit
├── go.mod / go.sum
├── AGENTS.md         # already written by this rectification — do not touch
├── tests/
│   └── work_schedule_test.go   # package tests
└── docs/
    ├── PLAN.md        # this file — delete once merged, per AGENTS.md
    ├── ARCHITECTURE.md
    ├── SKILL.md
    └── diagrams/database.md    # unchanged — no schema/ERD change in this plan
```

`mcp.go` and `mcp_test.go` are **deleted** (their content moves into `module.go`+`ops.go` and
`tests/work_schedule_test.go` respectively). There is no `view.go` and no `interfaces.go` — see
Stages 2 and 4 for why.

## Stage 0 — `go.mod`: remove the local fork

Delete this line from `go.mod`:

```
replace github.com/tinywasm/mcp => ../../../tinywasm/mcp
```

This alone does not make the module compile (the `mcp` import is still used by `mcp.go` until Stage
3) — it is listed first only because it is the standalone anti-pattern fix `AGENTS.md` calls out.
Land it together with Stage 3/7 in the same change; there is no reason to split it into its own
commit.

## Stage 1 — `model.go`: migrate to `model.Definition`

Bump dependencies first (exact versions — match `github.com/veltylabs/item_catalog`'s current
`go.mod`, do not guess a newer or older one):

```
go get github.com/tinywasm/model@v0.1.2 github.com/tinywasm/orm@v0.11.4 github.com/tinywasm/form@v0.3.13 github.com/tinywasm/router@v0.1.19
```

`model.Definition`/`model.Field`/`model.Kind` are explained in full in `AGENTS.md`'s whitelist
section and in `github.com/veltylabs/item_catalog/model.go` (read that file for the general shape if
anything below is unclear) — this stage does not re-derive that contract, only gives this module's
exact target.

### 1.1 — Two corrections versus this repo's own stale `docs/PLAN.md` draft (now superseded by this
file)

The previous draft of this plan (being replaced by this document) proposed keeping a hand-written
`TableName()` method on `Staff`/`WorkCalendar` alongside the new `Definition`. That is **unnecessary
and must NOT be carried over**: `model.Definition.Name` **is** "the model identity: table name,
`ModelName()`, route key" (verified against `github.com/tinywasm/model@v0.1.2/definition.go`) —
`ormc` generates `ModelName()` returning exactly `Definition.Name`, and `orm.DB.Create/Update/Delete/Query`
all key off `m.ModelName()` (verified against `orm@v0.11.4/db.go`), never a separate `TableName()`
concept. A hand-written `TableName()` next to a `Definition{Name: "workcalendar"}` would be dead code
that happens to agree with the generated method — drop it. Set `Definition.Name` to the exact legacy
table name for both models (`"staff"`, `"workcalendar"` — no underscore, matching the physical table)
and do not add any `TableName()` method to either struct.

### 1.2 — Rewrite `model.go` to exactly this (no build tag: `model`/`form/input` are isomorphic)

```go
package workschedule

import (
	"github.com/tinywasm/form/input"
	"github.com/tinywasm/model"
)

// Staff maps to the legacy 'staff' table, owned by an external system. READ-ONLY: this module
// never migrates or writes it — see AGENTS.md's domain notes. No widgets: nobody builds a form to
// edit legacy staff records (model_orm.go today has no Widget on any of its fields either).
var StaffModel = model.Definition{
	Name: "staff",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), NotNull: true},
		{Name: "role", Type: model.Text(), NotNull: true},
		{Name: "email", Type: model.Text(), DB: &model.FieldDB{Unique: true}},
		{Name: "password_hash", Type: model.Text(), Exclude: true}, // in struct, out of codec/DB scan
	},
}

// WorkCalendar maps to the legacy 'workcalendar' table, owned by an external system. READ-ONLY —
// same rule as Staff.
var WorkCalendarModel = model.Definition{
	Name: "workcalendar",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "staff_id", Type: model.Int(), NotNull: true}, // FK to staff, enforced at app layer only
		{Name: "day_of_week", Type: model.Int(), NotNull: true},
		{Name: "start_time", Type: model.Text(), NotNull: true},
		{Name: "end_time", Type: model.Text(), NotNull: true},
		{Name: "is_active", Type: model.Bool(), NotNull: true},
	},
}

// The 3 structs below are transport-only (args/response of get_work_schedule) — DB: nil.
// Widget policy is by ROLE, not "what today's model_orm.go had" (that file put widgets on every
// transport field — a defect this migration corrects, not a split to preserve): input.X() ONLY on
// user-editable fields (here: the one args field a caller fills in); base kinds (model.X()) on the
// two RESPONSE models — output is never rendered as an editable form, and a widget there would
// make form.New produce editable inputs for data the user must not touch.

var GetWorkScheduleArgsModel = model.Definition{
	Name: "get_work_schedule_args",
	Fields: model.Fields{
		{Name: "staff_id", Type: input.Number()},
	},
}

var ScheduleEntryModel = model.Definition{
	Name: "schedule_entry",
	Fields: model.Fields{
		{Name: "day", Type: model.Int()},
		{Name: "day_name", Type: model.Text()},
		{Name: "is_active", Type: model.Bool()},
		{Name: "start", Type: model.Text(), OmitEmpty: true},
		{Name: "end", Type: model.Text(), OmitEmpty: true},
	},
}

var StaffResponseModel = model.Definition{
	Name: "staff_response",
	Fields: model.Fields{
		{Name: "staff_name", Type: model.Text()},
		{Name: "staff_role", Type: model.Text()},
		{Name: "schedule", Type: model.StructSlice(&ScheduleEntryModel)},
	},
}
```

Notes preserved from the old analysis (still correct, keep this reasoning in mind while checking
`ormc`'s output, do not re-litigate it):

- `PasswordHash`: `Exclude: true` puts the field on the generated `Staff` struct but out of
  `Pointers()`/`EncodeFields`/`DecodeFields` — exactly today's `db:"-"` behavior. `tests/work_schedule_test.go`
  (Stage 6) still sets `Staff{PasswordHash: "hash123"}` as a plain struct literal; that keeps compiling
  because the field still exists on the struct, only the codec ignores it.
- `WorkCalendar.staff_id`: the old `db:"ref=staff,not_null"` tag fed an `ormc` FK-helper mechanism this
  module never used (verified: no `ReadAllWorkCalendarByStaffID`-style call exists anywhere in this
  repo; the actual filter is `.Where(WorkCalendar_.StaffId).Eq(staffId)` in the query builder). Do not
  set `Field.Ref` on `staff_id` — just `NotNull: true`, as written above.
- `StaffResponseModel.schedule`: the nested `Definition` is a parameter of the **Kind constructor**
  (`model.StructSlice(&ScheduleEntryModel)`), never a separate `Field.Ref` on that field — setting both
  is a contradiction `ormc` rejects at generation time.
- Casing is pure Go casing throughout the regenerated struct: `id`→`Id`, `staff_id`→`StaffId` (never
  `ID`/`StaffID`). Every consumer in `module.go`/`ops.go`/`tests/` must use the new casing.
- `DayOfWeek`/`Day` were `int` (32-bit) in the old struct; `model.Int()`'s fixed mapping is `int64`, so
  they become `int64` in the regenerated struct. `dayNames[7]string` indexing needs an explicit
  `int(...)` conversion where the compiler requires it (see Stage 3's `module.go`).

### 1.3 — Regenerate `model_orm.go`

```
go install github.com/tinywasm/ormc/cmd/ormc@latest
ormc   # run from the module root
```

Because `orm.DB.Query`/`Create`/`Update`/`Delete` take a `model.Model` (not just `model.Fielder`) at
this API version, the regenerated file will also emit `EncodeFields`/`DecodeFields`/`IsNil` for
`Staff`, `WorkCalendar`, and the 3 transport structs — none of that existed in the old
`model_orm.go` (which predates the `model.Encodable`/`Decodable` split). This is expected; do not try
to hand-write it, `ormc` does it. For the `schedule` field specifically, expect (verified against
`github.com/tinywasm/ormc`'s generator — `FieldStructSlice` with a non-pointer element resolves to a
value slice, not a pointer slice):

```go
type StaffResponse struct {
	StaffName string
	StaffRole string
	Schedule  []ScheduleEntry   // value slice — NOT []*ScheduleEntry
}
```

and, inside `EncodeFields`/`DecodeFields`, an `Array`/`Object` round-trip over that slice (the exact
lines are generated — do not hand-write them; if `ormc`'s output for this field looks materially
different from a `w.Array("schedule", len(m.Schedule))` / `r.Array("schedule")` loop, stop and report
it rather than hand-patching the generated file).

## Stage 2 — Identity: no `model.IDGenerator`

**This module needs no `Deps` struct and no `model.IDGenerator`.** Verified: nothing in the current
`mcp.go`/`model.go` ever constructs a `Staff`, `WorkCalendar`, or any other row — the module only
issues `ReadOneStaff`/`ReadAllWorkCalendar` queries. `unixid` is today only an indirect dependency
(pulled in transitively, not imported by this module's own code — `grep -rn unixid *.go` in this repo
returns nothing) confirming it was never actually used here. Do not add `Deps{IDs model.IDGenerator}`
to `module.go` — that would be inventing a need this module does not have. (See `AGENTS.md`'s domain
notes for the same statement, already recorded there.)

## Stage 3 — Transport: `router.OpModule` replaces `mcp.ToolProvider`

Delete `mcp.go`. Create `module.go` (the service) and `ops.go` (the transport binding) — no build tag
on either (`orm`/`router`/`model` are isomorphic per `AGENTS.md`'s Build Tags Rule).

### 3.1 — `module.go`

```go
package workschedule

import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/orm"
)

// ErrStaffNotFound is returned by GetWorkSchedule when no staff row matches the given id.
var ErrStaffNotFound = fmt.Err("staff", "not", "found")

// Module is a read-only adapter over two legacy tables ('staff', 'workcalendar') this module does
// not own — see AGENTS.md's domain notes. It mints no IDs and publishes no events, so it takes no
// Deps: New only needs the already-connected *orm.DB.
type Module struct {
	db *orm.DB
}

// New wires the module to an already-connected *orm.DB (backed by whatever storage.Conn the app
// chose). It never migrates a schema — see Stage 5 — and never fails, so it returns *Module, not
// (*Module, error): there is nothing here that can go wrong at construction time.
func New(db *orm.DB) *Module {
	return &Module{db: db}
}

// dayNames: known, accepted duplication — business_hours carries the same table in its view.
// Two copies is the tolerated maximum; if a third module needs day names (or i18n arrives), the
// glue moves upstream (tinywasm/time or the app) per the "glue is written once" lego rule — noted
// here so the decision is recorded, not silently forked again.
var dayNames = [7]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

// GetWorkSchedule returns the given staff member's weekly schedule, or ErrStaffNotFound.
func (m *Module) GetWorkSchedule(staffId int64) (StaffResponse, error) {
	staffRow := &Staff{}
	staffRow, err := ReadOneStaff(m.db.Query(staffRow).Where(Staff_.Id).Eq(staffId), staffRow)
	if err != nil {
		// Never swallow a real DB failure into "not found" — only orm.ErrNotFound maps to the
		// domain sentinel; anything else surfaces as the internal error it is (a DB outage
		// reported as "staff not found" is the silent failure the harness forbids).
		if err == orm.ErrNotFound {
			return StaffResponse{}, ErrStaffNotFound
		}
		return StaffResponse{}, err
	}
	if staffRow == nil {
		return StaffResponse{}, ErrStaffNotFound
	}

	calRows, err := ReadAllWorkCalendar(
		m.db.Query(&WorkCalendar{}).
			Where(WorkCalendar_.StaffId).Eq(staffId).
			OrderBy(WorkCalendar_.DayOfWeek).Asc(),
	)
	if err != nil {
		return StaffResponse{}, err
	}

	entries := make([]ScheduleEntry, len(calRows))
	for i, r := range calRows {
		dayIndex := int(r.DayOfWeek)
		if dayIndex < 0 || dayIndex > 6 {
			dayIndex = 0
		}
		e := ScheduleEntry{Day: r.DayOfWeek, DayName: dayNames[dayIndex], IsActive: r.IsActive}
		if r.IsActive {
			e.Start = r.StartTime
			e.End = r.EndTime
		}
		entries[i] = e
	}
	return StaffResponse{StaffName: staffRow.Name, StaffRole: staffRow.Role, Schedule: entries}, nil
}
```

### 3.2 — `ops.go`

```go
package workschedule

import (
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
)

const OpGetWorkSchedule = "get_work_schedule"

func (m *Module) ModelName() string { return "work_schedule" }

func (m *Module) MountOps(reg router.OpRegistry) {
	reg.Op(OpGetWorkSchedule, m.opGetWorkSchedule).
		Requires("work_schedule", model.Read).
		Accepts(&GetWorkScheduleArgs{})
}

var _ router.OpModule = (*Module)(nil)

func (m *Module) opGetWorkSchedule(ctx router.Context) {
	var args GetWorkScheduleArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	resp, err := m.GetWorkSchedule(args.StaffId)
	if err != nil {
		// Status convention (ecosystem-wide): 404 = not found, 500 = genuine internal error
		// only — never collapse both (the "runtime mystery" the harness forbids).
		if err == ErrStaffNotFound {
			ctx.WriteStatus(404)
			return
		}
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&resp); err != nil {
		ctx.WriteStatus(500)
	}
}
```

Behavior change versus today, intentional and acceptable — call it out in the PR description, do not
"fix" it: the old `mcp.go` used `req.Bind(&args)`, which rejected a missing/zero `staff_id` as
`"params invalid"` before ever touching the DB. `router.Context.Decode` performs no such
NotNull-style validation — a zero/missing `staff_id` now simply falls through to `ErrStaffNotFound`
(no staff has id 0) and comes back as a **404** (the ecosystem status convention: 400
decode/validation, 404 not-found, 500 genuine internal errors only — never collapsed). A malformed
body still yields 400 via `Decode`. If the composition root later wants a distinct "missing
staff_id" 400, that is a follow-up, not part of this migration.

The single MCP tool (`get_work_schedule`, param `staff_id`, response
`staff_name`/`staff_role`/`schedule[]`) is preserved 1:1 as the single op `get_work_schedule` — same
name, same argument, same response shape.

## Stage 4 — View: no `view.go`

**Judgment call, state it explicitly rather than deciding silently:** this module does **not** get a
`NewView(caller router.Caller) view.Presenter`. `view.New(...)` is built around a list/select/save/
delete shape (`listOp` + a `model.FielderSlice` projected into `[]view.Item` for a picker, optionally
`WithSaveOp`/`WithDeleteOp`). `get_work_schedule` is a single detail lookup keyed by `staff_id` that
returns one nested composite value (`StaffResponse{StaffName, StaffRole, Schedule []ScheduleEntry}`)
— there is no `OpList*` here, nothing to select from a list, nothing to save, nothing to delete. Forcing
this into `view.New`'s shape would mean inventing a fake "list of one" with no product behind it. Do
not add `view.go`/`NewView` to this module. If a future requirement needs a UI for this schedule (a
detail panel, not a list), that is a new, separate design question — not something this plan should
paper over by bolting on an ill-fitting `view.Presenter` today.

## Stage 5 — No schema-migration stage (explicit, do not add one)

This module's `New()` must **never** call `ddl.CreateTable`/`ddl.Sync` (or the removed
`orm.DB.CreateTable`) for `Staff` or `WorkCalendar`. Both tables are owned and migrated by an external
legacy system; this module only ever issues `ReadOneStaff`/`ReadAllWorkCalendar` queries against them.
This is the one module in the whole `veltylabs/modules/*` batch where "the module owns its schema
migration" does **not** apply — the schema belongs to someone else. Do not import
`github.com/tinywasm/ddl` at all in this repo's non-test code (Stage 7 keeps it out of `go.mod`
entirely, not even as an indirect dependency worth commenting on) — there is nothing here that needs
it. If a future reviewer suggests adding a `ddl.CreateTable` call to "match the other modules," that
suggestion is wrong for this module specifically; point them at this stage and at `AGENTS.md`'s
domain notes, which say the same thing.

## Stage 6 — Tests: move to `tests/`, build over `storage/mem`

Delete `mcp_test.go`. Create `tests/work_schedule_test.go`, package `tests` (external — exercises only
the exported API). The test builds its own `*orm.DB` via `orm.New(mem.New())` — this `*orm.DB` value
is owned by the test itself (not the module's private field), so seeding the legacy tables directly on
it (`db.Create(&workschedule.Staff{...})`) is legitimate: it is exactly the same `*orm.DB` value the
test then also hands to `workschedule.New(db)` to build the module under test. This resolves what would
otherwise be a contradiction (a read-only module whose own tests need to seed data): the test is not
calling any write method the module doesn't have — `orm.DB.Create` is a public method the test calls
directly on the connection it constructed itself, same as `db.CreateTable`/`db.Create` in the old
`mcp_test.go`, just against `storage/mem` instead of a concrete `sqlite` handle.

```go
package tests

import (
	"testing"

	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router/mock"
	"github.com/tinywasm/storage/mem"
	workschedule "github.com/veltylabs/work_schedule"
)

func TestGetWorkSchedule_ValidStaff(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)

	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General",
		Email: "ana@example.com", PasswordHash: "hash123",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}
	calendars := []*workschedule.WorkCalendar{
		{Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true}, // Monday
		{Id: 2, StaffId: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true}, // Wednesday
		{Id: 3, StaffId: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},          // Friday
	}
	for _, c := range calendars {
		if err := db.Create(c); err != nil {
			t.Fatalf("seed work calendar: %v", err)
		}
	}

	resp, err := m.GetWorkSchedule(1)
	if err != nil {
		t.Fatalf("GetWorkSchedule: %v", err)
	}
	if resp.StaffName != "Dra. Ana González" {
		t.Errorf("expected staff_name %q, got %q", "Dra. Ana González", resp.StaffName)
	}
	if resp.StaffRole != "Médico General" {
		t.Errorf("expected staff_role %q, got %q", "Médico General", resp.StaffRole)
	}
	if len(resp.Schedule) != 3 {
		t.Fatalf("expected 3 schedule entries, got %d", len(resp.Schedule))
	}
	expectedNames := []string{"Lunes", "Miércoles", "Viernes"}
	for i, e := range resp.Schedule {
		if e.DayName != expectedNames[i] {
			t.Errorf("entry %d: expected day name %q, got %q", i, expectedNames[i], e.DayName)
		}
		if e.IsActive {
			if e.Start == "" || e.End == "" {
				t.Errorf("entry %d: expected active entry to have start/end times", i)
			}
		} else if e.Start != "" || e.End != "" {
			t.Errorf("entry %d: expected inactive entry to omit start/end times", i)
		}
	}
}

func TestGetWorkSchedule_StaffNotFound(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General", Email: "ana@example.com",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	if _, err := m.GetWorkSchedule(99); err != workschedule.ErrStaffNotFound {
		t.Fatalf("expected ErrStaffNotFound, got %v", err)
	}
}

func TestMountOps_RoutesAndEnforcesRBAC(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General", Email: "ana@example.com",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	reg := &mock.Router{}
	m.MountOps(reg)

	// No Authorize configured: mock.Config's zero value denies every guarded route.
	denied := &mock.Context{}
	denied.SetUserID("u1")
	denied.InBody = []byte(`{"staff_id":1}`)
	reg.Invoke("OP", "/get_work_schedule", denied)
	if denied.Status != 403 {
		t.Fatalf("expected 403 with no Authorize configured, got %d", denied.Status)
	}

	// Configure an Authorize that allows everything, then retry.
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})
	ok := &mock.Context{}
	ok.SetUserID("u1")
	ok.InBody = []byte(`{"staff_id":1}`)
	reg.Invoke("OP", "/get_work_schedule", ok)
	if ok.Status != 0 { // handler never calls WriteStatus on success
		t.Fatalf("expected no error status, got %d", ok.Status)
	}
	if len(ok.ResponseBody()) == 0 {
		t.Fatalf("expected an encoded response body")
	}
}

func TestOpGetWorkSchedule_InvalidParam(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)

	reg := &mock.Router{}
	m.MountOps(reg)
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})

	ctx := &mock.Context{}
	ctx.SetUserID("u1")
	ctx.InBody = []byte(`not valid json`)
	reg.Invoke("OP", "/get_work_schedule", ctx)
	if ctx.Status != 400 {
		t.Fatalf("expected 400 for a malformed body, got %d", ctx.Status)
	}
}
```

**Verified by actually running this against the real `tinywasm/json` codec, not assumed:**
`{"staff_id": "not-a-number"}` does **not** produce a decode error with this codec — a JSON string
where a number is expected is read as an empty numeric token and silently decodes to `0`, so the
handler proceeds to `GetWorkSchedule(0)` and returns **404** (`ErrStaffNotFound`), not 400. The
codec only errors on a structurally invalid top-level body (doesn't start with `{`, or is empty) —
confirmed by testing `not valid json`, `` (empty), and `[1,2,3]`, all of which correctly 400 via
`json.Decode`'s own `expected object, got ...` check, while `{"staff_id": "not-a-number"}` and even
a truncated `{"staff_id": 1` both silently succeeded with a 404. Use `not valid json` (or any body
not starting with `{`) to exercise the actual decode-error path — a per-field type mismatch on a
numeric field is not a way to trigger 400 in this ecosystem's codec.

Two tests from the old `mcp_test.go` do **not** carry over — drop them, do not try to force an
equivalent:

- `TestGetWorkSchedule_MissingParam` — depended on `req.Bind`'s NotNull enforcement, which no longer
  exists at the transport boundary (see Stage 3's behavior-change note). A missing/zero `staff_id` is
  now covered by `TestGetWorkSchedule_StaffNotFound`'s same code path.
- `TestGetWorkSchedule_DBFailure` — simulated a DB failure via `db.DropTable(&Staff{})`. `DropTable`
  now lives in `github.com/tinywasm/ddl` (verified against `orm@v0.11.4/docs/ARQUITECTURE.md`), which
  this module must never import (Stage 5). There is no in-repo way to simulate this failure without
  violating this module's own no-DDL rule; it added no coverage beyond the not-found/invalid-param
  paths already tested above.

## Stage 7 — `go.mod` end state

Remove entirely as a **direct** requirement: `github.com/tinywasm/mcp` (+ its `replace` line),
`github.com/tinywasm/sqlite`, `github.com/tinywasm/json`, `github.com/tinywasm/context` (no handler
needs it once `mcp.Request`/`*context.Context` are gone — `router.Context` replaces both roles).

**Expect `go mod tidy` to re-add `github.com/tinywasm/json` as an indirect (`// indirect`) require**
(verified by actually building `tests/work_schedule_test.go` against the real published
dependencies) — pulled in transitively via `tinywasm/router/mock`'s own codec use, not imported by
any file in this module. This is correct and expected; do not try to "fix" it by re-adding a
`replace` or hand-editing it out of `go.sum`. The acceptance grep below is already scoped to `.go`
files for exactly this reason — it checks what this module's own code imports, not `go.sum`'s
transitive closure.

Do **not** add: `github.com/tinywasm/ddl` (Stage 5 — this module never calls it, so it must not even
be an indirect line worth keeping around), `github.com/tinywasm/view`/`github.com/tinywasm/events`
(Stage 4/2 — unused), any `Deps` type pulling in `unixid`.

Bump: `github.com/tinywasm/fmt`, `github.com/tinywasm/form`, add `github.com/tinywasm/model`,
`github.com/tinywasm/orm`, `github.com/tinywasm/router` — all pinned to whatever
`github.com/veltylabs/item_catalog`'s `go.mod` currently has (as of this revision: `fmt v0.25.5`,
`form v0.3.13`, `model v0.1.2`, `orm v0.11.4`, `router v0.1.19` — re-check item_catalog's `go.mod` at
execution time in case it moved further; that repo, not this plan, is the live source of truth for
versions).

```
module github.com/veltylabs/work_schedule

go 1.25.2

require (
	github.com/tinywasm/fmt v0.25.5
	github.com/tinywasm/form v0.3.13
	github.com/tinywasm/model v0.1.2
	github.com/tinywasm/orm v0.11.4
	github.com/tinywasm/router v0.1.19
)
```

`github.com/tinywasm/storage` (for `storage/mem`, imported directly by `tests/work_schedule_test.go`)
and `github.com/tinywasm/router/mock` will be added automatically by `go mod tidy` as direct
requirements once those imports exist — do not hand-add version numbers for them, let `go mod tidy`
resolve them against the `router`/`orm` versions above. Run `go mod tidy` after all `.go` changes land
and let it settle every indirect line (including dropping `github.com/tinywasm/unixid`, which nothing
will import anymore).

## Stage 8 — Delete `docs/CHECK_PLAN.md`

Its content is fully superseded by this file (Stage 1's model migration covers everything
`CHECK_PLAN.md` described, at current rather than stale versions). Delete it.

## Stage 9 — Update `docs/SKILL.md` and `docs/ARCHITECTURE.md`

Both are rewritten as part of this same rectification (already done in this repo, doc-only, ahead of
this plan's dispatch) to describe the target `MountOps`/`module.go`/`ops.go` shape instead of
`RegisterTools`/`mcp.go`. Verify after implementing that they still match what you actually built —
if `ormc`'s output or a handler signature ended up different from what this plan describes, fix the
docs to match reality, do not leave them describing the plan's intent instead of the shipped code.

## Acceptance criteria

```bash
# No forbidden imports anywhere, tests included:
grep -rn "tinywasm/mcp\|tinywasm/json\|tinywasm/unixid\|tinywasm/sqlite\|tinywasm/sqlt\|tinywasm/postgres\|tinywasm/context\|tinywasm/layout" . --include='*.go'
# → empty

# The replace directive is gone:
grep -n "replace" go.mod
# → empty

# No ddl import anywhere (this module never migrates a schema):
grep -rn "tinywasm/ddl" . --include='*.go'
# → empty

# No hand-written TableName() method left over:
grep -rn "func.*TableName" . --include='*.go'
# → empty

# router.OpModule is satisfied:
grep -n "var _ router.OpModule" ops.go
# → present

# Tests live under tests/, package tests:
head -1 tests/work_schedule_test.go
# → "package tests"

# No view.go, no Deps/IDGenerator invented:
test ! -f view.go && echo "no view.go, as intended"
grep -rn "model.IDGenerator\|type Deps" . --include='*.go'
# → empty

gotest ./...
# → green, both !wasm and wasm (GOOS=js GOARCH=wasm) targets, per AGENTS.md's testing contract
```

## Stages table

| # | Stage | Output | Criterion |
|---|---|---|---|
| 0 | Remove `replace` line | `go.mod` | folds into 3/7; verified by acceptance grep |
| 1 | `model.go` → `model.Definition` | 5 Definitions, `TableName()` dropped, `ormc` regenerated | compiles, `Staff`/`WorkCalendar` casing pure (`Id`, `StaffId`) |
| 2 | Identity | no `Deps`, no `model.IDGenerator` | `grep -rn model.IDGenerator` empty |
| 3 | Transport | `module.go` + `ops.go`, `mcp.go` deleted | `var _ router.OpModule` compiles; `get_work_schedule` op registered |
| 4 | View decision | no `view.go` (documented reason) | `test ! -f view.go` |
| 5 | No schema migration | `New()` unchanged in spirit — no `ddl` call | `grep -rn tinywasm/ddl` empty |
| 6 | Tests | `tests/work_schedule_test.go`, `storage/mem` | `gotest ./...` green |
| 7 | `go.mod` end state | mcp/sqlite/json/context/unixid gone | acceptance grep block empty |
| 8 | Delete `docs/CHECK_PLAN.md` | file removed | `test ! -f docs/CHECK_PLAN.md` |
| 9 | Verify `docs/SKILL.md`/`docs/ARCHITECTURE.md` | match shipped code | manual review |
