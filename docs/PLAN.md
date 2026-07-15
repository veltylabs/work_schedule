# PLAN — work_schedule: migrar model.go a model.Definition

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

✅ **Desbloqueado.** `github.com/tinywasm/model@v0.0.14` (con `orm@v0.9.28`) ya lee `model.Definition` y
soporta `model.Field.Exclude`. `go get github.com/tinywasm/model@v0.0.14 github.com/tinywasm/orm@v0.9.28`
antes de regenerar. ⚠️ **Casing puro:** `id`→`Id`, `staff_id`→`StaffId` (ya no `ID`/`StaffID`);
actualiza esas referencias en `mcp.go`/tests (ver §5).

Eres un agente **sin contexto previo** y **solo tienes este repositorio** (`work_schedule`). Plan
autocontenido: todo contrato, regla y ejemplo está inline.

---

## 1. Qué cambia y por qué

El ecosistema tinywasm invirtió la generación de modelos: se escribe una definición tipada
(`model.Definition`) a mano, y `ormc` genera el struct concreto + plomería. Migración **mecánica**:
mismo comportamiento, mismas tablas legadas (`staff`, `workcalendar`), sin tocar DDL.

## 2. Contrato de `github.com/tinywasm/model` (inline)

`Field.Type` **no** es un literal de un enum — es la interfaz `Kind`. Se rellena llamando a un
constructor (`model.Text()`, `model.Int()`, …), nunca asignando `model.FieldText` directamente. La
composición (`FieldStruct`/`FieldStructSlice`) **tampoco** usa `Field.Ref`: el `*Definition` anidado
es ahora un parámetro obligatorio del constructor del Kind (`model.Struct(ref)`/
`model.StructSlice(ref)`) — poner `Field.Ref` en un campo de composición es **error de generación**
en `ormc` (ver §4, campo `schedule`).

```go
package model

// FieldType es el mapeo determinista de almacenamiento/wire — lo devuelve Kind.Storage(),
// nunca se asigna directamente a Field.Type.
type FieldType int
const (
    FieldText FieldType = iota // string
    FieldInt                   // int64
    FieldFloat                 // float64
    FieldBool                  // bool
    FieldBlob                  // []byte
    FieldStruct                // struct anidado — Kind = model.Struct(ref)
    FieldIntSlice               // []int
    FieldStructSlice            // []T anidado — Kind = model.StructSlice(ref)
    FieldRaw                    // JSON pre-serializado
)

// Kind reemplaza el antiguo par Field.Type-enum + Field.Widget. Implementaciones
// sin estado, seguras para reuso concurrente.
type Kind interface {
    Storage() FieldType          // mapeo determinista a Go/DDL
    Name() string                // nombre semántico: "text", "int", "email", ...
    Validate(value string) error // SIEMPRE presente — fail-closed
}

// Constructores base — devuelven Kind, no un literal FieldType:
func Text() Kind                       // storage FieldText
func Int() Kind                        // storage FieldInt
func Float() Kind                      // storage FieldFloat
func Bool() Kind                       // storage FieldBool
func Blob() Kind                       // storage FieldBlob
func Struct(ref *Definition) Kind      // storage FieldStruct — ref es parámetro obligatorio
func StructSlice(ref *Definition) Kind // storage FieldStructSlice — ídem

type FieldDB struct { PK, Unique, AutoInc bool }

type Field struct {
    Name      string
    Type      Kind        // model.Text(), model.Int(), ... — NUNCA un literal FieldType
    NotNull   bool
    OmitEmpty bool        // omitir del JSON si es zero value
    DB        *FieldDB    // nil = sin persistencia
    Ref       *Definition // SOLO FK escalar; en FieldStruct/FieldStructSlice el ref va en el
                          // constructor del Kind (model.Struct(ref)), NO aquí
    Exclude   bool        // campo en el struct generado, PERO fuera de Pointers()/codec —
                          // úsalo para datos que el struct debe llevar sin que ormc los toque
    Permitted
}

type Fields = []Field

type Definition struct {
    Name   string
    Fields Fields
}
```

Mapeo fijo: `model.Text()`→`string`, `model.Int()`→`int64`, `model.Bool()`→`bool`. Variable de
definición debe llamarse `<Struct>Model`.

**Ya no existe `Field.Widget`.** Un Kind con UI es un `Kind` de `github.com/tinywasm/form/input`
(p. ej. `input.Text()`). Este módulo **sí** usa widgets hoy — en 3 de sus 5 `Definition` (ver
§4) — así que no basta con los Kinds base para todo el archivo.

---

## 3. Estado actual (`model.go`, a portar)

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

**Estado READ-ONLY / sin DDL:** esta propiedad no está impuesta hoy por ningún flag de `ormc` — se
cumple porque el código de este módulo **nunca** llama `db.Create`/migración para `Staff`/
`WorkCalendar`; solo usa `ReadOneX`/`ReadAllX`. Se preserva automáticamente con esta migración: no
añadas ninguna llamada de creación/migración para estas dos tablas.

**Dos hallazgos de esta migración (revisar §4 abajo):**

1. **`PasswordHash string \`db:"-"\`** — hoy existe en el struct pero se excluye del schema/codec (se
   usa vía otro canal; ver `mcp_test.go`). Bajo el nuevo flujo, el struct se **deriva** de la
   `Definition` — no hay forma de que un campo exista en el struct sin estar en `Fields`. Se resuelve
   con `Exclude: true` (§2): el campo entra en `Fields`, `ormc` lo emite en el `struct` pero lo omite
   de `Pointers()`/codec, preservando el comportamiento actual exactamente.

2. **`StaffID int64 \`db:"ref=staff,not_null"\`** — el `ref=staff` alimenta hoy un mecanismo interno
   de `ormc` para generar un helper `ReadAllXByStaffID` (relación FK). **Verificado: este módulo no
   usa ese helper** — el filtrado por `staff_id` se hace con `qb.Query(...).Where(WorkCalendar_
   .StaffId).Eq(staffID)` en `mcp.go` (casing puro: `staff_id`→`StaffId`), no con un helper generado.
   Se **elimina** la anotación de
   relación en la migración (queda solo `NotNull: true`); no hay pérdida de comportamiento observable.
   Si en el futuro se necesita un helper `ByStaffID`, se puede añadir explícitamente sin depender de
   esta anotación.

## 4. Estado objetivo (`model.go` reescrito)

```go
//go:build !wasm

package workschedule

import (
	"github.com/tinywasm/form/input"
	"github.com/tinywasm/model"
)

// Staff/WorkCalendar: sin widgets — el `model_orm.go` ACTUAL tampoco los tiene en
// ninguno de sus campos (son READ-ONLY, nadie construye un form para editarlos).
// No inventes widgets aquí: preserva ese estado.
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

func (s *Staff) TableName() string { return "staff" }

// WorkCalendar maps to the legacy 'workcalendar' table. READ-ONLY — no DDL allowed.
var WorkCalendarModel = model.Definition{
	Name: "workcalendar",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "staff_id", Type: model.Int(), NotNull: true}, // FK relation to staff, enforced at app layer
		{Name: "day_of_week", Type: model.Int(), NotNull: true},
		{Name: "start_time", Type: model.Text(), NotNull: true},
		{Name: "end_time", Type: model.Text(), NotNull: true},
		{Name: "is_active", Type: model.Bool(), NotNull: true},
	},
}

func (w *WorkCalendar) TableName() string { return "workcalendar" }

// Los 3 structs de abajo SÍ tienen widget hoy en el `model_orm.go` actual (campo
// `Widget:` de la API vieja) — son los args/respuesta que arman la vista del
// horario. Preserva esa asignación exacta con `input.X()`.

var GetWorkScheduleArgsModel = model.Definition{
	Name: "get_work_schedule_args",
	Fields: model.Fields{
		{Name: "staff_id", Type: input.Number()},
	},
}

var ScheduleEntryModel = model.Definition{
	Name: "schedule_entry",
	Fields: model.Fields{
		{Name: "day", Type: input.Number()},
		{Name: "day_name", Type: input.Text()},
		{Name: "is_active", Type: input.Checkbox()},
		{Name: "start", Type: input.Text(), OmitEmpty: true},
		{Name: "end", Type: input.Text(), OmitEmpty: true},
	},
}

var StaffResponseModel = model.Definition{
	Name: "staff_response",
	Fields: model.Fields{
		{Name: "staff_name", Type: input.Text()},
		{Name: "staff_role", Type: input.Text()},
		{Name: "schedule", Type: model.StructSlice(&ScheduleEntryModel)},
	},
}
```

**Nota de migración de tipo:** `DayOfWeek`/`Day` eran `int` (32-bit); con el mapeo fijo `FieldInt` →
`int64` pasan a `int64` en el struct generado. Revisa comparaciones/aritmética en `mcp.go` /
`buildStaffResponse` que asuman `int` — ajusta a `int64` donde el compilador lo exija.

**Por qué estos 3 sí y `Staff`/`WorkCalendar` no:** verificado contra el `model_orm.go` que este
mismo repo tiene generado *hoy*: `Staff`/`WorkCalendar` no tienen ningún `Widget:` asignado, pero
`GetWorkScheduleArgsModel`, `ScheduleEntryModel` y `StaffResponseModel` sí, campo por campo,
exactamente como quedó arriba. Dejarlos como Kinds base sin widget rompería en silencio el form
que arma la vista del horario — el mismo defecto ya corregido en `service_catalog`.

**Nota de composición (`StaffResponseModel.schedule`):** el ref anidado va en el **constructor** del
Kind — `model.StructSlice(&ScheduleEntryModel)` —, **no** en un `Field.Ref` separado. Poner ambos
(`Type: model.FieldStructSlice, Ref: &ScheduleEntryModel`, como haría una traducción mecánica del
enum viejo) es una contradicción que `ormc` rechaza con error de generación: `Field.Ref` es solo para
FK escalares.

## 5. Pasos

> **Dependencias:** `go get github.com/tinywasm/model@v0.0.14 github.com/tinywasm/orm@v0.9.28 github.com/tinywasm/form@v0.2.15`
> (`model` directa nueva, antes solo se llegaba transitivamente vía `orm`; `form` ya era
> dependencia directa (v0.2.6) — se bumpea para regenerar los 3 widgets de §4).

1. Reescribe `model.go` con el contenido de §4 (conserva `TableName()` en ambos structs — son métodos
   escritos a mano, no generados; Go permite declarar métodos de un tipo en otro archivo del mismo
   paquete).
2. Regenera `model_orm.go` con `ormc` (instalado/actual), sin directivas. Verifica:
   - `Staff` conserva `PasswordHash string` en el struct, pero NO en `Pointers()` ni en
     `EncodeFields`/`DecodeFields` generados (si existen para este tipo).
   - ⚠️ **Casing puro:** `WorkCalendar.StaffID`→`StaffId`, `Staff.ID`→`Staff.Id` (sigue `int64`/`string`),
     sin el helper `ByStaffID` (no existía realmente en uso).
   - `DayOfWeek`/`Day` son `int64`.
3. Ajusta `mcp.go` y `mcp_test.go`: referencias `.StaffID`→`.StaffId`, `.ID`→`.Id`; y cualquier
   construcción de `WorkCalendar{...}`/`scheduleEntry{...}` con literales `int` para `DayOfWeek`/`Day`
   debe usar `int64` (o dejar que Go infiera si son constantes sin tipo). Columnas/JSON no cambian.
4. Verifica que `mcp_test.go` (que ya construye `Staff{PasswordHash: "hash123"}`) siga compilando: el
   campo sigue existiendo en el struct.

## 6. Fuera de alcance

- No tocar el esquema de las tablas legadas `staff`/`workcalendar` (son de solo lectura).
- No añadir un helper `ReadAllWorkCalendarByStaffID` — no se usaba y no forma parte de esta migración.
- No añadir widgets **nuevos** que no tuviera ya el `model_orm.go` actual (no le pongas widget a
  `Staff`/`WorkCalendar`: hoy no lo tienen). Sí **preservar** los 3 que ya existen (§4).

## 7. Criterio de aceptación

- `gotest ./...` verde con `go.mod` en `model v0.0.14` / `orm v0.9.28` / `form v0.2.15`.
- `Staff.PasswordHash` existe en el struct generado y sigue siendo asignable en tests, pero no
  aparece en el `Pointers()` generado ni en el codec.
- Casing puro aplicado (`Id`, `StaffId`) en todos los consumidores; `DayOfWeek`/`Day` son `int64`.
- `GetWorkScheduleArgsModel`, `ScheduleEntryModel`, `StaffResponseModel` conservan sus widgets
  (`input.Number()`/`input.Text()`/`input.Checkbox()`, ver §4); `Staff`/`WorkCalendar` siguen sin
  ninguno.
- No queda struct plano con tags `db:` ni directiva en `model.go`.
- `StaffResponseModel.schedule` usa `Type: model.StructSlice(&ScheduleEntryModel)` — sin `Field.Ref`
  puesto por separado en ese campo.

## 8. Etapas

| # | Etapa | Salida | Criterio |
|---|---|---|---|
| 1 | Reescribir `model.go` | Definitions de §4 (incl. `Exclude`, sin `ref=`; 3 structs conservan sus widgets `input.X()`, `Staff`/`WorkCalendar` sin widget) | compila (ormc actualizado) |
| 2 | Regenerar `model_orm.go` | struct + plomería; `PasswordHash` excluido del codec | inspección manual conforme |
| 3 | Ajustar `int64` en callers | `mcp.go`/`mcp_test.go` actualizados | `gotest ./...` verde |
