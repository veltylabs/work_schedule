# Arquitectura de work-schedule

## 1. Alcance del Dominio
Proporciona una vista de solo lectura del horario laboral semanal para el personal clínico (médicos, enfermeras, etc.).
Lee de dos tablas legadas mantenidas por un sistema externo — este módulo nunca es propietario de su esquema ni lo migra.

## 2. Entidades Principales
- **Staff:** El empleado de la clínica.
- **WorkCalendar:** Una fila por miembro del personal por día laboral. `is_active = false` indica
  un día no laboral; `start_time`/`end_time` solo tienen sentido cuando `is_active = true`.

## 3. Patrones Arquitectónicos
Este módulo está acoplado solo a contratos publicados, no a infraestructura concreta — ver
`AGENTS.md` (en la raíz de este repositorio) para la lista completa permitida/prohibida. Aplicado aquí:

1. **Solo Lectura / Sin DDL:** `New(db *orm.DB) *Module` nunca llama a `ddl.Sync` para `Staff`/`WorkCalendar`. Las tablas son manejadas completamente por el sistema legado. Este es el único módulo en el lote `veltylabs/modules/*` donde "el módulo maneja su propia migración de esquema" **no** aplica.
2. **Inyección de Dependencias, sin `Deps`:** `New(db *orm.DB) *Module` toma solo el `*orm.DB` ya conectado.
3. **Transporte — `crud.Module`:** `ModelName() string` + `MountOperations(reg crud.Registry)`
   registran las operaciones CRUD del módulo, apoyándose en la infraestructura autogenerada y transaccional de `crud.Registry`.
4. **`view.Presenter` — lista de solo lectura (una forma):** `get_work_schedule` (vía `work_schedule_r`) es una búsqueda de detalle única indexada por `staff_id`. El presenter del módulo (`NewView(caller, staffId)`) es una vista **solo para lista/selección** sobre `ScheduleEntry`. Cada `ScheduleEntry` es proyectada sobre `Item()` (generado por ormc).
5. **Agnóstico de Codificación:** las estructuras implementan `model.Encodable`/`Decodable` generados por `ormc`.

## 4. Operaciones (vía `MountOperations`)
Las operaciones usan ayudantes autogenerados en `crud.Registry`.
- `work_schedule_r`: Lee el horario anidado de un empleado dado su `staff_id` como filtro.

## 5. Ejemplo de Raíz de Composición
```go
sched := workschedule.New(db) // *orm.DB — sin Deps, sin migración de esquema
sched.MountOperations(crudRegistry) // crud.Registry
pres := workschedule.NewView(caller, staffID) // solo lista/selección, sin Saver/Deleter
```
