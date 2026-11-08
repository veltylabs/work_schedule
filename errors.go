package workschedule

import "webtyp.com/model"

// ErrStaffNotFound es devuelto por GetWorkSchedule cuando ninguna fila de personal coincide con el id dado.
//
// Ejemplo:
// 	if staff == nil { return ErrStaffNotFound }
const ErrStaffNotFound model.NotFound = "staff not found"

// ErrCannotRead es devuelto cuando la solicitud no puede ser decodificada.
//
// Ejemplo:
// 	if err := ctx.Decode(&args); err != nil { return ErrCannotRead }
const ErrCannotRead model.Invalid = "work_schedule: cannot read the request"
