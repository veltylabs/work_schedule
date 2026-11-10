# work-schedule
<img src="docs/img/badges.svg">

Adaptador de solo lectura para el horario laboral semanal del personal clínico sobre `crud`.
Lee de las tablas legadas `staff` y `workcalendar` — sin operaciones DDL.

## Operaciones

`definition.go` declara un solo modelo de lectura, `work_schedule` (`Allow: model.Read`); `crud` lo monta. Una operación es `router.Operation{Module: ModelName, Entity: "work_schedule", Action: model.Read}` con el filtro `{staff_id}` (requerido: sin él, la lectura se rechaza). Una persona que no existe es «no encontrado». `Staff` y `WorkCalendar` son tablas legadas: no se montan.

## Inicio Rápido

```go
import workschedule "github.com/veltylabs/work-schedule"

m := workschedule.New(db)  // solo lectura: sin creación de tablas
// crud monta el modelo de lectura que declara definition.go:
crud.Mount(reg, nil, nil, m)

// Vista de lista de solo lectura sobre el horario aplanado de un miembro del personal (en browser):
pres := workschedule.NewView(caller, staffID)
```

`NewView` es **solo para lista/selección**; este módulo nunca escribe en sus tablas legadas — el *editor* del horario vive en `appointment_booking`. Consulta `docs/ARCHITECTURE.md`.
La lista carga según lo que declara su `view.Page` (`view.Page{First: view.All}`) y se busca en el servidor usando `crud.ReadArgs`.

## Documentación

| Documento | Descripción |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Alcance del dominio, patrones y referencia |
| [Diagrama de Base de Datos](docs/diagrams/database.md) | Diagrama de esquema legado |
| [SKILL.md](docs/SKILL.md) | Resumen condensado amigable para LLM |
