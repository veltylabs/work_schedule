package workschedule

import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/orm"
)

// ErrStaffNotFound is returned by GetWorkSchedule when no staff row matches the given id.
var ErrStaffNotFound = fmt.Err("staff", "not", "found")

// Module is a read-only adapter over two legacy tables ('staff', 'workcalendar') this module does
// not own — see AGENTS.md's domain notes. It mints no IDs and publishes no events, so it takes no
// Deps: New only needs the already-connected *orm.DB.
type Module struct {
	db *orm.DB
}

// New wires the module to an already-connected *orm.DB (backed by whatever storage.Conn the app
// chose). It never migrates a schema — see Stage 5 — and never fails, so it returns *Module, not
// (*Module, error): there is nothing here that can go wrong at construction time.
func New(db *orm.DB) *Module {
	return &Module{db: db}
}

// dayNames: known, accepted duplication — business_hours carries the same table in its view.
// Two copies is the tolerated maximum; if a third module needs day names (or i18n arrives), the
// glue moves upstream (tinywasm/time or the app) per the "glue is written once" lego rule — noted
// here so the decision is recorded, not silently forked again.
var dayNames = [7]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

// GetWorkSchedule returns the given staff member's weekly schedule, or ErrStaffNotFound.
func (m *Module) GetWorkSchedule(staffId int64) (StaffResponse, error) {
	staffRow := &Staff{}
	staffRow, err := ReadOneStaff(m.db.Query(staffRow).Where(Staff_.Id).Eq(staffId), staffRow)
	if err != nil {
		// Never swallow a real DB failure into "not found" — only orm.ErrNotFound maps to the
		// domain sentinel; anything else surfaces as the internal error it is (a DB outage
		// reported as "staff not found" is the silent failure the harness forbids).
		if err == orm.ErrNotFound {
			return StaffResponse{}, ErrStaffNotFound
		}
		return StaffResponse{}, err
	}
	if staffRow == nil {
		return StaffResponse{}, ErrStaffNotFound
	}

	calRows, err := ReadAllWorkCalendar(
		m.db.Query(&WorkCalendar{}).
			Where(WorkCalendar_.StaffId).Eq(staffId).
			OrderBy(WorkCalendar_.DayOfWeek).Asc(),
	)
	if err != nil {
		return StaffResponse{}, err
	}

	entries := make([]ScheduleEntry, len(calRows))
	for i, r := range calRows {
		dayIndex := int(r.DayOfWeek)
		if dayIndex < 0 || dayIndex > 6 {
			dayIndex = 0
		}
		e := ScheduleEntry{Day: r.DayOfWeek, DayName: dayNames[dayIndex], IsActive: r.IsActive}
		if r.IsActive {
			e.Start = r.StartTime
			e.End = r.EndTime
		}
		entries[i] = e
	}
	return StaffResponse{StaffName: staffRow.Name, StaffRole: staffRow.Role, Schedule: entries}, nil
}
