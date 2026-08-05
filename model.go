package workschedule

import (
	"github.com/tinywasm/form/input"
	"github.com/tinywasm/model"
)

// Staff se mapea a la tabla heredada 'staff', propiedad de un sistema externo. SÓLO LECTURA: este módulo
// nunca realiza migraciones ni escribe en ella — ver notas de dominio en AGENTS.md. Sin widgets: nadie construye
// un formulario para editar registros de personal heredados (model_orm.go hoy tampoco tiene Widget en ninguno de sus campos).
var StaffModel = model.Definition{
	Name: "staff",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), NotNull: true},
		{Name: "role", Type: model.Text(), NotNull: true},
		{Name: "email", Type: model.Text(), DB: &model.FieldDB{Unique: true}},
		{Name: "password_hash", Type: model.Text(), Exclude: true}, // en la estructura, fuera del códec/escaneo de la BD
	},
}

// WorkCalendar se mapea a la tabla heredada 'workcalendar', propiedad de un sistema externo. SÓLO LECTURA —
// misma regla que Staff.
var WorkCalendarModel = model.Definition{
	Name: "workcalendar",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "staff_id", Type: model.Int(), NotNull: true}, // FK a staff, aplicada únicamente en la capa de la aplicación
		{Name: "day_of_week", Type: model.Int(), NotNull: true},
		{Name: "start_time", Type: model.Text(), NotNull: true},
		{Name: "end_time", Type: model.Text(), NotNull: true},
		{Name: "is_active", Type: model.Bool(), NotNull: true},
	},
}

// Las 3 estructuras de abajo son solo de transporte (argumentos/respuesta de get_work_schedule) — DB: nil.
// La política de widgets es por ROL, no "lo que model_orm.go tenía hoy" (ese archivo ponía widgets en cada
// campo de transporte — un defecto que esta migración corrige, no una división a conservar): input.X() SÓLO en
// campos editables por el usuario (aquí: el único campo de argumentos que completa un llamador); tipos base (model.X())
// en los dos modelos de RESPUESTA — la salida nunca se renderiza como un formulario editable, y un widget allí
// haría que form.New produjera entradas editables para datos que el usuario no debe tocar.

var GetWorkScheduleArgsModel = model.Definition{
	Name: "get_work_schedule_args",
	Fields: model.Fields{
		{Name: "staff_id", Type: input.Number()},
	},
}

var ScheduleEntryModel = model.Definition{
	Name: "schedule_entry",
	Fields: model.Fields{
		{Name: "day", Type: model.Int()},
		{Name: "day_name", Type: model.Text()},
		{Name: "is_active", Type: model.Bool()},
		{Name: "start", Type: model.Text(), OmitEmpty: true},
		{Name: "end", Type: model.Text(), OmitEmpty: true},
	},
}

var StaffResponseModel = model.Definition{
	Name: "staff_response",
	Fields: model.Fields{
		{Name: "staff_name", Type: model.Text()},
		{Name: "staff_role", Type: model.Text()},
		{Name: "schedule", Type: model.StructSlice(&ScheduleEntryModel)},
	},
}
