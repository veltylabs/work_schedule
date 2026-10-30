# work-schedule — Resumen de Habilidad LLM

## Propósito
Adaptador de solo lectura sobre las tablas legadas `staff` y `workcalendar`, propiedad de un sistema externo.
Expone una única operación (`get_work_schedule`) que retorna el horario semanal de un miembro del personal.

## Archivos Clave
| Archivo | Rol |
|------|------|
| `model.go` | `StaffModel`/`WorkCalendarModel`/`GetWorkScheduleArgsModel`/`ScheduleEntryModel`/`StaffResponseModel` (literales `model.Definition`) |
| `model_orm.go` | Autogenerado por `ormc` — NO EDITAR |
| `module.go` | `Module`, `New(db)`, `GetWorkSchedule(staffId)` — el servicio |
| `ops.go` | `ModelName()`, `MountOps(reg router.OpRegistry)` — la vinculación de transporte |
| `tests/work_schedule_test.go` | Todas las pruebas (paquete `tests`, `orm.New(mem.New())`) |

## Restricciones
- **Sin DDL, nunca.** Nunca llames a `ddl.Sync` (el eliminado `orm.DB.CreateTable` ya no existe)
  para `staff` o `workcalendar` — este módulo no es propietario de su esquema. Ver notas de dominio en `AGENTS.md`; no "corrijas" esto añadiendo una llamada de migración.
- **Sin `model.IDGenerator`, sin `Deps`.** Este módulo nunca construye una fila.
- **Sin `view.Presenter`.** Una búsqueda de detalle única, no una lista — ver `docs/ARCHITECTURE.md` §3.4.
- `password_hash` tiene `Exclude: true` en `StaffModel` — existe en la estructura generada, pero el codec ORM nunca la lee o escribe.
- `staff_id` llega vía `router.Context.Decode(&args)` — sin validación más allá de lo que `Decode` hace por sí mismo; un `staff_id` faltante/cero surge como `ErrStaffNotFound`, no como un "parámetro inválido" separado.
- `start`/`end` se omiten de `ScheduleEntry` cuando `is_active = false` (`OmitEmpty: true`).

## Registro de Transporte
```go
sched := workschedule.New(db)   // *orm.DB, sin Deps
sched.MountOps(opRegistry)      // router.OpRegistry — ej. mcp.HarvestOps(sched, ...)
```
