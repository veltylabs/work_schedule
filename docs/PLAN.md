---
PLAN: "test: work_schedule translate comments to Spanish, add nested-array roundtrip test"
TAG: v0.0.6
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

# PLAN — work_schedule: comentarios en español + prueba de roundtrip anidado

Eres un agente **sin contexto previo** y **solo tienes este repositorio** (`work_schedule`). El
módulo ya adoptó el patrón del arnés reutilizable en una ronda anterior; este plan es un ajuste
pequeño y autocontenido sobre ese trabajo ya mergeado.

## 1. Por qué existe este plan

Dos cosas quedaron pendientes de la ronda anterior:

1. **Comentarios todavía en inglés** en `module.go` y `ops.go` — el resto del batch (`business_hours`,
   `provider_payouts`) ya tradujo los suyos a español, manteniendo el código (identificadores) en
   inglés. Este módulo no lo hizo.
2. **Cobertura 27.8%**, medida con `go test ./tests/... -coverpkg=github.com/veltylabs/work_schedule
   -cover`. El objetivo del ecosistema es >=80%, pero **la mayoría de lo que falta es plumbing
   generado estructuralmente inalcanzable en este módulo específico — ver §3, no lo persigas más
   allá de lo que este plan pide**.

## 2. Traducir comentarios a español — `module.go` y `ops.go`

Traduce **todos** los comentarios existentes en ambos archivos a español, sin tocar ningún
identificador, string literal de negocio (nombres de días, mensajes de error) ni lógica. Ejemplo del
patrón esperado (aplica el mismo criterio al resto de comentarios en ambos archivos):

```go
// ANTES
// New wires the module to an already-connected *orm.DB (backed by whatever storage.Conn the app
// chose). It never migrates a schema — see Stage 5 — and never fails, so it returns *Module, not
// (*Module, error): there is nothing here that can go wrong at construction time.
func New(db *orm.DB) *Module {

// DESPUÉS
// New conecta el módulo a un *orm.DB ya conectado (respaldado por el storage.Conn que la app haya
// elegido). Nunca migra un esquema — ver Etapa 5 — y nunca falla, por lo que devuelve *Module, no
// (*Module, error): no hay nada aquí que pueda salir mal en el momento de la construcción.
func New(db *orm.DB) *Module {
```

Aplica el mismo criterio a: el comentario de `ErrStaffNotFound`, el docstring de `Module`, el
comentario de `dayNames`, el docstring de `GetWorkSchedule`, el comentario dentro de su manejo de
error (`// Never swallow a real DB failure...`), y el comentario de convención de estado dentro de
`opGetWorkSchedule` en `ops.go` (`// Status convention (ecosystem-wide)...`).

## 3. Analizado — por qué 27.8%→~43% es el techo razonable aquí, no 80%

Este módulo es un adaptador de **solo lectura** sobre dos tablas legadas (`staff`, `workcalendar`)
que no posee (ver `AGENTS.md`, notas de dominio). Eso significa que aproximadamente la mitad de los
métodos generados por `ormc` para `Staff`/`WorkCalendar` **nunca se invocan por ningún camino real de
este módulo**:

- `Staff.EncodeFields`/`DecodeFields`/`Validate`, `WorkCalendar.EncodeFields`/`DecodeFields`/
  `Validate` — estos tipos son solo de lectura de DB (`Pointers()` sí se usa, vía el escaneo interno
  del ORM); nunca viajan directamente por el wire (solo sus DTOs derivados `StaffResponse`/
  `ScheduleEntry` lo hacen), y este módulo nunca crea/actualiza ninguno de los dos (no es dueño del
  esquema — Etapa 5).
- `ReadAllStaff`, `ReadOneWorkCalendar` — generados automáticamente para todo modelo con rol de DB,
  pero este módulo nunca los llama (usa `ReadOneStaff`/`ReadAllWorkCalendar`, no las variantes
  opuestas).
- `StaffList`, `WorkCalendarList`, `GetWorkScheduleArgsList`, `ScheduleEntryList`,
  `StaffResponseList` — ninguno de estos 5 modelos se usa jamás como lista de nivel superior (el
  único arreglo real es `StaffResponse.Schedule []ScheduleEntry`, vía `model.StructSlice`, un
  mecanismo distinto al de estos wrappers `FielderSlice`). Boilerplate generado sin uso.
- `ModelName()`/`Schema()`/`Pointers()` en `GetWorkScheduleArgs`/`ScheduleEntry`/`StaffResponse` —
  el codec JSON usa `EncodeFields`/`DecodeFields` directamente, nunca estos tres; solo importarían si
  este módulo construyera un `form.New(...)` sobre alguno de ellos, lo cual no hace.

**No escribas pruebas para cerrar estos huecos** — serían pruebas que llaman un método generado
directamente solo para mover el contador de cobertura, sin probar ningún comportamiento real de este
módulo (exactamente el antipatrón de "cobertura inflada" que `AGENTS.md` señala). Si una ronda futura
decide que el objetivo de cobertura debe aplicarse literalmente incluso aquí, es una decisión de
alcance distinta — repórtala, no la implementes especulativamente.

## 4. La única prueba de valor real que falta — roundtrip del arreglo anidado

`TestMountOps_RoutesAndEnforcesRBAC` (en `tests/work_schedule_test.go`) siembra un `Staff` pero
**ningún** `WorkCalendar` — por lo tanto `StaffResponse.Schedule` siempre queda vacío en esa prueba,
y el único camino que de verdad ejercita la codificación del arreglo anidado
(`StaffResponse.Schedule []ScheduleEntry`, vía `model.StructSlice` — `arr.Object(&x)` en
`StaffResponse.EncodeFields`, y el loop simétrico en `DecodeFields`) nunca se recorre en ningún test
existente. Esto es real riesgo sin probar, no relleno: es la parte más compleja/frágil de este
módulo y hoy nadie la verifica de punta a punta.

**Modifica `TestMountOps_RoutesAndEnforcesRBAC`** en `tests/work_schedule_test.go`:

1. Después de sembrar el `Staff` (antes de construir `reg := &mock.Router{}`), agrega una fila de
   `WorkCalendar` y una aserción de `ModelName`:

```go
	// Al menos una fila de calendario — sin esto, StaffResponse.Schedule queda vacío y el
	// roundtrip de codificación anidada (model.StructSlice) nunca se ejercita de verdad.
	if err := db.Create(&workschedule.WorkCalendar{
		Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true,
	}); err != nil {
		t.Fatalf("seed work calendar: %v", err)
	}

	if m.ModelName() != "work_schedule" {
		t.Fatalf("expected ModelName %q, got %q", "work_schedule", m.ModelName())
	}
```

2. Al final de la función, justo después de `if len(ok.ResponseBody()) == 0 { t.Fatalf(...) }`,
   agrega (requiere importar `"github.com/tinywasm/json"` — ver §5):

```go
	// Decodifica la respuesta a través del codec real — prueba que el arreglo anidado
	// (model.StructSlice, StaffResponse.Schedule []ScheduleEntry) realmente serializa y
	// deserializa de punta a punta, no solo que el body no esté vacío.
	var got workschedule.StaffResponse
	if err := json.Decode(ok.ResponseBody(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.StaffName != "Dra. Ana González" {
		t.Errorf("expected decoded staff_name %q, got %q", "Dra. Ana González", got.StaffName)
	}
	if len(got.Schedule) != 1 {
		t.Fatalf("expected 1 decoded schedule entry, got %d", len(got.Schedule))
	}
	if got.Schedule[0].DayName != "Lunes" || got.Schedule[0].Start != "09:00" {
		t.Errorf("unexpected decoded schedule entry: %+v", got.Schedule[0])
	}
```

## 5. Import a agregar

En `tests/work_schedule_test.go`, agrega `"github.com/tinywasm/json"` al bloque de imports (junto a
`model`, `orm`, `router/mock`, `storage/mem`). Corre `go mod tidy` después — pasa a ser un require
directo de este archivo de test (la excepción documentada en `AGENTS.md`: *"in tests, use
`github.com/tinywasm/json` for codec verification"*).

## 6. Fuera de alcance

- No tocar `model.go`, `model_orm.go`, `ops.go` más allá de la traducción de comentarios de §2.
- No agregar pruebas para lo descartado en §3.
- No perseguir 80% — ~43% es el resultado esperado de aplicar exactamente §2+§4; es el techo
  razonable dado que este módulo es un adaptador de solo lectura sobre tablas que no posee.

## 7. Criterio de aceptación

- `go build ./...` y `GOOS=js GOARCH=wasm go build ./...` limpios.
- `gotest ./...` verde.
- `grep -n "^\s*//" module.go ops.go` — ningún comentario en inglés remanente.
- `go test ./tests/... -coverpkg=github.com/veltylabs/work_schedule -cover` reporta ~43% (no 80% —
  ver §3).
- `git status` limpio tras el commit.
