# work-schedule
<img src="docs/img/badges.svg">

Adaptador MCP de solo lectura para el horario laboral semanal del personal clínico.
Lee de las tablas legadas `staff` y `workcalendar` — sin operaciones DDL.

## Herramientas MCP

| Herramienta | Parámetros | Descripción |
|------|-----------|-------------|
| `get_work_schedule` | `staff_id` (número) | Devuelve el perfil del personal y el horario semanal. |

## Inicio Rápido

```go
import workschedule "github.com/veltylabs/work-schedule"

m := workschedule.New(db)  // solo lectura: sin creación de tablas
m.MountOperations(opReg)   // router.OperationRegistry — expone get_work_schedule

// Vista de lista de solo lectura sobre el horario aplanado de un miembro del personal:
pres := workschedule.NewView(caller, staffID)
```

`NewView` es **solo para lista/selección**; este módulo nunca escribe en sus tablas legadas — el *editor* del horario vive en `appointment_booking`. Consulta `docs/ARCHITECTURE.md`.

## Documentación

| Documento | Descripción |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Alcance del dominio, patrones y referencia de herramientas MCP |
| [Diagrama de Base de Datos](docs/diagrams/database.md) | Diagrama de esquema legado |
| [SKILL.md](docs/SKILL.md) | Resumen condensado amigable para LLM |

errores por `rpc.WriteError`; permisos por tabla (`model.ResourceOf`); claves foráneas declaradas.
