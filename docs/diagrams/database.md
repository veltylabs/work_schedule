# work-schedule — Diagrama de Base de Datos

```mermaid
flowchart TD
    S[staff — LEGADO SOLO-LECTURA]
    S --> S1[id: int64 PK]
    S --> S2[name: string NOT NULL]
    S --> S3[role: string NOT NULL]
    S --> S4[email: string UNIQUE]
    S --> S5[password_hash: string excluido de ORM]

    W[workcalendar — LEGADO SOLO-LECTURA]
    W --> W1[id: int64 PK]
    W --> W2[staff_id: int64 NOT NULL]
    W --> W3[day_of_week: int NOT NULL<br/>0=Domingo … 6=Sábado]
    W --> W4[start_time: string NOT NULL HH:MM]
    W --> W5[end_time: string NOT NULL HH:MM]
    W --> W6[is_active: bool NOT NULL]

    W2 --> S1
```

> **No se permite DDL.** Las tablas son propiedad del sistema legado.
> Estrategia de lectura: primero obtener `staff` por `staff_id`, luego obtener todas
> las filas de `workcalendar` para ese personal ordenadas por `day_of_week ASC`.
