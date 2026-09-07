package workschedule

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

const OpGetWorkSchedule = "get_work_schedule"

func (m *Module) ModelName() string { return "work_schedule" }

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpGetWorkSchedule, m.opGetWorkSchedule).
		Requires("work_schedule", model.Read).
		Accepts(&GetWorkScheduleArgs{})
}

var _ router.OperationModule = (*Module)(nil)

func (m *Module) opGetWorkSchedule(ctx router.Context) {
	var args GetWorkScheduleArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	resp, err := m.GetWorkSchedule(args.StaffId)
	if err != nil {
		// Convención de estado (en todo el ecosistema): 404 = no encontrado, 500 = error interno
		// real únicamente — nunca colapsar ambos (el "misterio en tiempo de ejecución" que el arnés prohíbe).
		if err == ErrStaffNotFound {
			ctx.WriteStatus(404)
			return
		}
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&resp); err != nil {
		ctx.WriteStatus(500)
	}
}
