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
		// Status convention (ecosystem-wide): 404 = not found, 500 = genuine internal error
		// only — never collapse both (the "runtime mystery" the harness forbids).
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
