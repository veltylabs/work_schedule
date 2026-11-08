package workschedule

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

// Staff maps to the legacy 'staff' table, owned by an external system. READ-ONLY: this module
// never migrates or writes it — see AGENTS.md's domain notes. No widgets: nobody builds a form to
// edit legacy staff records (model_orm.go today has no Widget on any of its fields either).
var StaffModel = model.Definition{
	Name: "staff",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), NotNull: true},
		{Name: "role", Type: model.Text(), NotNull: true},
		{Name: "email", Type: model.Text(), DB: &model.FieldDB{Unique: true}},
		{Name: "password_hash", Type: model.Text(), Exclude: true}, // in struct, out of codec/DB scan
	},
}

// WorkCalendar maps to the legacy 'workcalendar' table, owned by an external system. READ-ONLY —
// same rule as Staff.
var WorkCalendarModel = model.Definition{
	Name: "workcalendar",
	Fields: model.Fields{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "staff_id", Type: model.Int(), NotNull: true}, // FK to staff, enforced at app layer only
		{Name: "day_of_week", Type: model.Int(), NotNull: true},
		{Name: "start_time", Type: model.Text(), NotNull: true},
		{Name: "end_time", Type: model.Text(), NotNull: true},
		{Name: "is_active", Type: model.Bool(), NotNull: true},
	},
}

var GetWorkScheduleArgsModel = model.Definition{
	Name: "get_work_schedule_args",
	Fields: model.Fields{
		{Name: "staff_id", Type: input.Number()},
	},
}

var ScheduleEntryModel = model.Definition{
	Name: "schedule_entry",
	Fields: model.Fields{
		{Name: "day", Type: model.Int(), DB: &model.FieldDB{PK: true}},
		{Name: "day_name", Type: model.Text(), Item: model.ItemLabel},
		{Name: "is_active", Type: model.Bool(), Item: model.ItemDescription},
		{Name: "start", Type: model.Text(), OmitEmpty: true, Item: model.ItemDescription},
		{Name: "end", Type: model.Text(), OmitEmpty: true, Item: model.ItemDescription},
	},
}

var WorkScheduleModel = model.Definition{
	Name: "work_schedule",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "staff_name", Type: model.Text(), Item: model.ItemLabel},
		{Name: "staff_role", Type: model.Text(), Item: model.ItemDescription},
		{Name: "schedule", Type: model.StructSlice(&ScheduleEntryModel)},
	},
}
