package workschedule

import (
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
)

const OpGetWorkSchedule = "get_work_schedule"

func (m *Module) ModelName() string { return "work_schedule" }

func (m *Module) MountOps(reg router.OpRegistry) {
	reg.Op(OpGetWorkSchedule, m.opGetWorkSchedule).
		Requires("work_schedule", model.Read).
		Accepts(&GetWorkScheduleArgs{})
}

var _ router.OpModule = (*Module)(nil)

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
