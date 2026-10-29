package workschedule

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/rpc"
)

const OpGetWorkSchedule = "get_work_schedule"

func (m *Module) ModelName() string { return "work_schedule" }

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpGetWorkSchedule, m.opGetWorkSchedule).
		Requires(ResourceWorkSchedule, model.Read).
		Accepts(&GetWorkScheduleArgs{})
}

var _ router.OperationModule = (*Module)(nil)

func (m *Module) opGetWorkSchedule(ctx router.Context) {
	var args GetWorkScheduleArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	resp, err := m.GetWorkSchedule(args.StaffId)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&resp); err != nil {
		rpc.WriteError(ctx, err)
	}
}
