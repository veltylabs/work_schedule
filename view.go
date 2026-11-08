//go:build browser

package workschedule

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

type scheduleLister struct {
	view.Lister
	staffId int64
}

func (l scheduleLister) List(args crud.ReadArgs, done func([]view.Record, error)) {
	args.Filter = &GetWorkScheduleArgs{StaffId: l.staffId}

	// Intercept done to flatten the WorkSchedule -> ScheduleEntry records for list view
	l.Lister.List(args, func(rows []view.Record, err error) {
		if err != nil {
			done(nil, err)
			return
		}
		if len(rows) == 0 {
			done(nil, nil)
			return
		}
		ws, ok := rows[0].(*WorkSchedule)
		if !ok {
			done(nil, ErrCannotRead)
			return
		}

		entries := make([]view.Record, len(ws.Schedule))
		for i := range ws.Schedule {
			entries[i] = &ws.Schedule[i]
		}
		done(entries, nil)
	})
}

func NewView(caller router.Caller, staffId int64) view.Presenter {
	recs := view.NewRecords()
	baseLister := view.NewCallerLister(
		recs.Caller(caller),
		view.Ops{Module: "work_schedule", Allow: model.Read, Page: view.Page{First: view.All}},
		func() model.ModelSlice { return &WorkScheduleList{} },
	)
	return view.New(
		scheduleLister{Lister: baseLister, staffId: staffId},
		&ScheduleEntry{},
		view.WithTitle("Horario (sistema legado)"),
	)
}
