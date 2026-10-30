# Arquitectura de work-schedule

## 1. Alcance del Dominio
Proporciona una vista de solo lectura del horario laboral semanal para el personal clínico (médicos, enfermeras, etc.).
Lee de dos tablas legadas mantenidas por un sistema externo — este módulo nunca es propietario de su esquema ni lo migra.

## 2. Entidades Principales
- **Staff:** El empleado de la clínica. `password_hash` se excluye del escaneo DB / codec ORM mediante
  `model.Field{Exclude: true}` — aún existe en la estructura `Staff` generada (para el canal lateral que lo maneje), pero los métodos `Pointers()`/`EncodeFields`/`DecodeFields` generados por `ormc` nunca lo tocan.
- **WorkCalendar:** Una fila por miembro del personal por día laboral. `is_active = false` indica
  un día no laboral; `start_time`/`end_time` solo tienen sentido cuando `is_active = true`.

## 3. Patrones Arquitectónicos
Este módulo está acoplado solo a contratos publicados, no a infraestructura concreta — ver
`AGENTS.md` (en la raíz de este repositorio) para la lista completa permitida/prohibida. Aplicado aquí:

1. **Solo Lectura / Sin DDL:** `New(db *orm.DB) *Module` nunca llama a `ddl.Sync` (el eliminado `orm.DB.CreateTable` ya no existe) para `Staff`/`WorkCalendar`. Las tablas son manejadas completamente por el sistema legado. Este es el único módulo en el lote `veltylabs/modules/*` donde "el módulo maneja su propia migración de esquema" **no** aplica — el esquema pertenece a otro. `New()` no tiene efectos secundarios más que almacenar la referencia a `*orm.DB`.
2. **Inyección de Dependencias, sin `Deps`:** `New(db *orm.DB) *Module` toma solo el `*orm.DB` ya conectado. A diferencia de la mayoría de los módulos en este ecosistema, este no declara `Deps{IDs, Publisher}` — no genera IDs (nunca construye una fila `Staff`/`WorkCalendar`) y no publica eventos (una lectura no tiene nada que anunciar). No agregues ninguno especulativamente.
3. **Transporte — `router.OpModule`:** `ModelName() string` + `MountOps(reg router.OpRegistry)`
   registran la única operación del módulo con `.Requires(resource, action)` + `.Accepts(&ArgsType{})`.
   El módulo nunca implementa `router.APIModule`/`Router`, y nunca importa `tinywasm/mcp` — la aplicación decide qué transporte (`mcp.HarvestOps`, o cualquier futuro) cosecha la operación.
4. **`view.Presenter` — lista de solo lectura (una forma):** `get_work_schedule` es una búsqueda de detalle única indexada por `staff_id`, retornando un valor compuesto anidado — por lo que el presenter del módulo (`NewView(caller, staffId)`) es una vista **solo para lista/selección** sobre el arreglo aplanado `Schedule`. Cada `ScheduleEntry` es proyectada sobre `Item()` (generado por ormc). No hay `Saver`/`Deleter`: este módulo nunca escribe en sus tablas legadas. Una aplicación consumidora lo monta para el panel de "horario legado"; el *editor* de horario vive en `appointment_booking`.
5. **Agnóstico de Codificación:** las estructuras de solicitud/respuesta (`GetWorkScheduleArgs`, `ScheduleEntry`, `StaffResponse`) implementan `model.Encodable`/`Decodable` (generado por `ormc`); `router.Context.Decode`/`Encode` recorren esa interfaz con cualquier codec concreto que el transporte instale. El módulo nunca importa `tinywasm/json` directamente.

## 4. Operaciones (vía `MountOps`)
| Op | Acción | Recurso | Parámetros | Retorno |
|----|--------|----------|--------|---------|
| `get_work_schedule` | `r` | `work_schedule` | `staff_id` (int) | `staff_name`, `staff_role`, `schedule[]` (`day`, `day_name`, `is_active`, `start`, `end`) |

Nombres de días en español: Domingo, Lunes, Martes, Miércoles, Jueves, Viernes, Sábado.
Los campos `start`/`end` se omiten cuando `is_active = false`.

## 5. Ejemplo de Raíz de Composición
```go
sched := workschedule.New(db) // *orm.DB — sin Deps, sin migración de esquema
sched.MountOps(opRegistry)    // router.OpRegistry
// Vista de lista de solo lectura (Schedule aplanado) para un panel de "horario legado":
pres := workschedule.NewView(caller, staffID) // solo lista/selección, sin Saver/Deleter
```

## 6. Esquema
Ver [`docs/diagrams/database.md`](diagrams/database.md). Tanto `staff` como `workcalendar` son tablas legadas propiedad de un sistema externo — nunca se emite un DDL en su contra desde este módulo.
