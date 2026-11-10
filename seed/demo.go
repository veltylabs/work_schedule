//go:build wasm

package seed

import (
	workschedule "github.com/veltylabs/work_schedule"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/view"
)

func Demo() (view.Demo, error) {
	db := orm.New(mem.New())
	mod := workschedule.New(db)

	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General",
		Email: "ana@example.com", PasswordHash: "hash123",
	}); err != nil {
		return view.Demo{}, err
	}
	calendars := []*workschedule.WorkCalendar{
		{Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true},
		{Id: 2, StaffId: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true},
		{Id: 3, StaffId: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},
	}
	for _, c := range calendars {
		if err := db.Create(c); err != nil {
			return view.Demo{}, err
		}
	}

	return view.Demo{
		Label:    "work_schedule",
		Timezone: "America/Santiago",
		ActingAs: "u1",
		Modules:  []crud.Module{mod},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return []view.Screen{{
				ID:         "work_schedule",
				Label:      "Horario",
				Icon:       "schedule",
				Permission: model.ResourceOf(&workschedule.WorkScheduleModel),
				Presenter:  workschedule.NewView(d.Caller, 1),
			}}
		},
	}, nil
}
