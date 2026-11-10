package workschedule

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// scheduleLister lista las entradas del horario de una persona: pide el modelo de lectura work_schedule con el filtro
// staff_id y entrega sus entradas (días) como filas de la lista.
type scheduleLister struct {
	caller  router.Caller
	staffId int64
}

func (l *scheduleLister) Page() view.Page { return view.Page{First: view.All} }

func (l *scheduleLister) Records() *view.Records { return nil }

func (l *scheduleLister) List(_ crud.ReadArgs, done func([]view.Record, error)) {
	var list WorkScheduleList
	op := router.Operation{Module: ModelName, Entity: (&WorkSchedule{}).ModelName(), Action: model.Read}
	l.caller.Call(op, &crud.ReadArgs{Filter: &WorkSchedule{StaffId: l.staffId}}, &list, func(err error) {
		if err != nil {
			done(nil, err)
			return
		}
		if len(list) == 0 {
			done(nil, nil)
			return
		}
		entries := make([]view.Record, len(list[0].Schedule))
		for i := range list[0].Schedule {
			entries[i] = &list[0].Schedule[i]
		}
		done(entries, nil)
	})
}

// NewView construye el Presenter del horario de una persona: una fila por día, con su descripción.
//
// Ejemplo: p := workschedule.NewView(caller, 1)
func NewView(caller router.Caller, staffId int64) view.Presenter {
	return view.New(&scheduleLister{caller: caller, staffId: staffId}, &ScheduleEntry{},
		view.WithTitle("Horario (sistema legado)"))
}
