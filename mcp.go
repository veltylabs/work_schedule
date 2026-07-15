//go:build !wasm

package workschedule

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/orm"
)

type Module struct {
	db *orm.DB
}

func New(db *orm.DB) *Module { return &Module{db: db} }

// RegisterTools registers the module's tools with the MCP server.
func (m *Module) RegisterTools(srv *mcp.Server) {
	for _, t := range m.Tools() {
		srv.AddTool(t)
	}
}

// Tools implements mcp.ToolProvider.
func (m *Module) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "get_work_schedule",
			Description: "Returns the work calendar for a staff member.",
			Action:      'r',
			Execute:     m.GetWorkSchedule,
		},
	}
}

// GetWorkSchedule returns the work calendar for a staff member.
func (m *Module) GetWorkSchedule(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args getWorkScheduleArgs
	if err := req.Bind(&args); err != nil {
		return &mcp.Result{IsError: true, Content: fmt.Err("params", "invalid").Error()}, nil
	}
	staffID := args.StaffID

	staffModel := &Staff{}
	staffRow, err := ReadOneStaff(
		m.db.Query(staffModel).Where(Staff_.ID).Eq(staffID),
		staffModel,
	)
	if err != nil || staffRow == nil {
		return &mcp.Result{IsError: true, Content: fmt.Err("staff", "not", "found").Error()}, nil
	}

	calRows, err := ReadAllWorkCalendar(
		m.db.Query(&WorkCalendar{}).
			Where(WorkCalendar_.StaffID).Eq(staffID).
			OrderBy(WorkCalendar_.DayOfWeek).Asc(),
	)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}

	resp := buildStaffResponse(staffRow, calRows)
	var s string
	if err := json.Encode(&resp, &s); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(s), nil
}

var dayNames = [7]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

func buildStaffResponse(s *Staff, rows []*WorkCalendar) staffResponse {
	entries := make([]scheduleEntry, len(rows))
	for i, r := range rows {
		dayIndex := r.DayOfWeek
		if dayIndex < 0 || dayIndex > 6 {
			dayIndex = 0 // fallback or handle error if needed, but safe bounds
		}
		e := scheduleEntry{
			Day:      r.DayOfWeek,
			DayName:  dayNames[dayIndex],
			IsActive: r.IsActive,
		}
		if r.IsActive {
			e.Start = r.StartTime
			e.End   = r.EndTime
		}
		entries[i] = e
	}
	return staffResponse{StaffName: s.Name, StaffRole: s.Role, Schedule: entries}
}
