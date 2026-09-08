# work-schedule
<img src="docs/img/badges.svg">

Read-only MCP adapter for the weekly work schedule of clinic staff.
Reads from legacy `staff` and `workcalendar` tables — no DDL operations.

## MCP Tools

| Tool | Parameters | Description |
|------|-----------|-------------|
| `get_work_schedule` | `staff_id` (number) | Returns staff profile and weekly schedule. |

## Quick Start

```go
import workschedule "github.com/veltylabs/work-schedule"

m := workschedule.New(db)  // read-only: no table creation
m.MountOperations(opReg)   // router.OperationRegistry — exposes get_work_schedule

// Read-only list view over a staff member's flattened schedule:
pres := workschedule.NewView(caller, staffID)
```

`NewView` is **list/select-only** (each `ScheduleEntry` is a `view.Item`); this
module never writes its legacy tables — the schedule *editor* lives in
`appointment_booking`. See `docs/ARCHITECTURE.md`.

## Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Domain scope, patterns, and MCP tool reference |
| [Database Diagram](docs/diagrams/database.md) | Legacy schema diagram |
| [SKILL.md](docs/SKILL.md) | LLM-friendly condensed summary |
