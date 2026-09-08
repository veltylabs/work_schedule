package workschedule

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// Item proyecta una ScheduleEntry como fila de lista. view.Itemizer.
// Desc: "HH:MM–HH:MM" si está activo, "Inactivo" si no.
func (e *ScheduleEntry) Item() view.Item {
	desc := "Inactivo"
	if e.IsActive {
		desc = e.Start + "–" + e.End
	}
	return view.Item{ID: e.DayName, Label: e.DayName, Description: desc}
}

// scheduleLister es un view.Lister a medida: llama get_work_schedule con el
// staff dado y aplana la respuesta a filas de ScheduleEntry. No hay op de lista
// plana (get_work_schedule devuelve StaffResponse anidada), así que no se usa
// view.NewCallerLister (que manda args nil y espera una lista).
type scheduleLister struct {
	caller  router.Caller
	staffId int64
}

func (l scheduleLister) List() ([]model.Model, error) {
	resp := &StaffResponse{}
	ch := make(chan error, 1)
	l.caller.Call(
		OpGetWorkSchedule,
		&GetWorkScheduleArgs{StaffId: l.staffId},
		resp,
		func(err error) { ch <- err },
	)
	if err := <-ch; err != nil {
		return nil, err
	}
	rows := make([]model.Model, 0, len(resp.Schedule))
	for i := range resp.Schedule {
		rows = append(rows, &resp.Schedule[i])
	}
	return rows, nil
}

var _ view.Lister = scheduleLister{}

// NewView construye un Presenter de solo lectura del horario semanal de UN
// profesional. Sin Saver/Deleter: este módulo nunca escribe sobre las tablas
// legadas (ver AGENTS.md). La edición de agenda vive en appointment_booking.
func NewView(caller router.Caller, staffId int64) view.Presenter {
	return view.New(
		scheduleLister{caller: caller, staffId: staffId},
		&ScheduleEntry{},
		view.WithTitle("Horario (sistema legado)"),
	)
}
