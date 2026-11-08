package workschedule

import (
	"webtyp.com/crud"
	"webtyp.com/fmt"
	"webtyp.com/model"
)

func (m *Module) readWorkSchedule(c crud.Caller, args crud.ReadArgs, filter model.Model) (WorkScheduleList, error) {
	staffFilter, ok := filter.(*GetWorkScheduleArgs)
	if !ok || staffFilter == nil {
		return nil, ErrCannotRead
	}
	resp, err := m.GetWorkSchedule(staffFilter.StaffId)
	if err != nil {
		return nil, err
	}
	resp.Id = fmt.Convert(staffFilter.StaffId).String()
	var list WorkScheduleList
	list = append(list, resp)
	return list, nil
}
