package workschedule

import (
	"webtyp.com/router"
	"webtyp.com/view"
)

type scheduleLister struct {
	records *view.Records
	caller  router.Caller
	staffId int64
}

func (l scheduleLister) Records() *view.Records { return l.records }

func (l scheduleLister) List(done func([]view.Record, error)) {
	resp := &StaffResponse{}
	l.caller.Call(
		OpGetWorkSchedule,
		&GetWorkScheduleArgs{StaffId: l.staffId},
		resp,
		func(err error) {
			if err != nil {
				done(nil, err)
				return
			}
			rows := make([]view.Record, 0, len(resp.Schedule))
			for i := range resp.Schedule {
			    copy := resp.Schedule[i]
				rows = append(rows, &copy)
			}
			done(rows, nil)
		},
	)
}

var _ view.Lister = scheduleLister{}

func NewView(caller router.Caller, staffId int64) view.Presenter {
	recs := view.NewRecords()
	return view.New(
		scheduleLister{records: recs, caller: recs.Caller(caller), staffId: staffId},
		&ScheduleEntry{},
		view.WithTitle("Horario (sistema legado)"),
	)
}
