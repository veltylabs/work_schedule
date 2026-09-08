---
PLAN: "feat(view): read-only NewView over get_work_schedule"
TAG: v0.2.0
EXECUTOR: local
REVIEWER: none
---

# PLAN — `work_schedule` vista solo-lectura (Etapa C2 del `DEMO_AGENDA_MASTER_PLAN`)

Orquestador: `webtyp/docs/DEMO_AGENDA_MASTER_PLAN.md` §7 fila C2.
(Copia local: `/home/cesar/Dev/Project/webtyp/docs/DEMO_AGENDA_MASTER_PLAN.md`.)

> **Prioridad baja / opcional.** El valor real del "módulo que le falta la UI en
> la demo" es el editor de agenda, que vive en `appointment_booking` (Etapa C) —
> `work_schedule` es un adaptador **solo-lectura** sobre tablas legadas y NO se
> vuelve editable (su `AGENTS.md` lo prohíbe). Esta etapa solo le da a la demo
> una vista de ejemplo consistente con `item_catalog.NewView`, para el panel
> "horario actual (legado)" del módulo demo. Si el tiempo aprieta, se difiere:
> el módulo demo `work_schedule` (Etapa D) puede montar directamente el editor
> de `appointment_booking` sin este panel.

## Contexto

`work_schedule` expone solo `get_work_schedule` → `StaffResponse{StaffName,
StaffRole, Schedule []ScheduleEntry}` (`ScheduleEntry{Day, DayName, IsActive,
Start, End}`, ≤7). No tiene `NewView`. Para que la demo lo consuma con el mismo
patrón que los demás módulos reales (`<mod>.NewView(caller, …)`), agregar una
vista de lista solo-lectura sobre las entradas del horario.

## Cambios (`view.go`, nuevo)

Importa solo `view` + `model` + `router` (whitelist).

```go
// Item proyecta una ScheduleEntry como fila de lista (view.Itemizer).
func (e *ScheduleEntry) Item() view.Item {
    desc := "Inactivo"
    if e.IsActive {
        desc = e.Start + "–" + e.End
    }
    return view.Item{ID: e.DayName, Label: e.DayName, Description: desc}
}

// NewView construye un Presenter de solo lectura del horario semanal de UN
// profesional. Sin Saver/Deleter: este módulo nunca escribe (ver AGENTS.md).
func NewView(caller router.Caller, staffId int64) view.Presenter
```

`NewView` usa un `view.Lister` a medida (no `view.NewCallerLister`, que manda
args nil y espera una op de lista plana): llama `get_work_schedule` con
`GetWorkScheduleArgs{StaffId: staffId}`, decodifica en `StaffResponse`, y
aplana `.Schedule` a `[]model.Model` (`*ScheduleEntry`). Mismo espíritu que
`appointment_booking/lister.go` (`reservationLister`).

`view.WithTitle("Horario (sistema legado)")`.

`ScheduleEntry` **ya implementa `model.Model`** en `model_orm.go` (tiene
`ModelName`/`Schema`/`Pointers`/`IsNil`/`EncodeFields`/`DecodeFields`/`Validate`,
y existe `ScheduleEntryList` con `Len`/`At`/`Append`) — es parte de
`StaffResponseModel` vía `model.StructSlice`. Usarlo directo; lo único a agregar
en `view.go` es el método `Item()` (`view.Itemizer`).

## Tests (`tests/`, `gotest`)

- `TestNewView_ListsScheduleEntries` — con un `router.Caller` doble que devuelve
  un `StaffResponse` de 3 entradas (2 activas, 1 inactiva), el `Presenter.List()`
  devuelve 3 `view.Item` con `Description` correcta ("08:00–12:00" / "Inactivo").
- No romper los tests existentes de `GetWorkSchedule`.

## Criterios de aceptación

- `gotest ./...` verde.
- `GOOS=js GOARCH=wasm go build ./...` OK.
- `README.md` menciona `NewView`; `docs/ARCHITECTURE.md` nota que la vista es
  solo-lectura por diseño y que la edición de agenda vive en
  `appointment_booking`.

## Fuera de alcance

- Cualquier escritura. Cualquier UI (es de `app-demo`).
