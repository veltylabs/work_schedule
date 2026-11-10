# Mapa de pruebas

| Test | Regla |
|---|---|
| [TestNewView_ListsScheduleEntries](../tests/view_test.go) | marcador: el presenter solo se prueba en el navegador |
| [TestGetWorkSchedule_ValidStaff](../tests/work_schedule_test.go) | el horario de una persona trae su nombre, rol y un día por fila del calendario, con la hora solo si el día está activo |
| [TestGetWorkSchedule_StaffNotFound](../tests/work_schedule_test.go) | una persona que no existe → `ErrStaffNotFound` (no encontrado) |
| [TestErrStaffNotFoundMessage](../tests/work_schedule_test.go) | el texto del error de persona no encontrada no cambia |
| [TestWorkSchedule_ReadByStaff](../tests/work_schedule_test.go) | leer `work_schedule` con filtro `staff_id` → un horario con su nombre, rol y entradas |
| [TestWorkSchedule_NoFilterIsError](../tests/work_schedule_test.go) | sin `staff_id` en el filtro (o sin filtro) la lectura se rechaza sobre `staff_id` |
| [TestWorkSchedule_UnknownStaffIsNotFound](../tests/work_schedule_test.go) | leer el horario de una persona inexistente → no encontrado |
| [TestWorkSchedule_NoWriteOperations](../tests/work_schedule_test.go) | `work_schedule.create` no existe: «unknown operation» |
| [TestMount_MatchesDefinition](../tests/work_schedule_test.go) | `crud.Mount` acepta la definición del módulo |
| [TestItemsDeclaredOnFields](../tests/work_schedule_test.go) | `ScheduleEntry.Item()` arma etiqueta, descripción e id desde los campos |
| [TestDemo_Valid](../tests/demo_test.go) | `seed.Demo()` es válida y trae al menos una pantalla (destino wasm) |
