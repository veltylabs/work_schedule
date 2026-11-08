package workschedule

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (m *Module) ModelName() string { return "work_schedule" }

func (m *Module) MountOperations(reg crud.Registry) {
	MountWorkScheduleRead(reg, func() model.Model { return &GetWorkScheduleArgs{} }, m.readWorkSchedule)
}

var _ crud.Module = (*Module)(nil)
