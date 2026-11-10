package workschedule

import (
	"webtyp.com/crud"
	"webtyp.com/fmt"
	"webtyp.com/model"
)

// Read es la regla de negocio de leer work_schedule: el horario semanal de la persona que trae filter.StaffId.
// Sin persona (o con id 0) rechaza la lectura; una persona que no existe es ErrStaffNotFound.
func (p *WorkSchedule) Read(a crud.CurrentUser, filter *WorkSchedule, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	if filter == nil || filter.StaffId == 0 {
		return nil, &model.FieldError{Field: FieldStaffId, Err: ErrCannotRead}
	}
	schedule, err := m.GetWorkSchedule(filter.StaffId)
	if err != nil {
		return nil, err
	}
	schedule.StaffId = filter.StaffId
	schedule.Id = fmt.Convert(filter.StaffId).String()
	list := WorkScheduleList{schedule}
	return &list, nil
}
