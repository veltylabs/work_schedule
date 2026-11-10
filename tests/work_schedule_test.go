package tests

import (
	workschedule "github.com/veltylabs/work_schedule"
	"strings"
	"testing"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/storage/mem"
)

func TestGetWorkSchedule_ValidStaff(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)

	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General",
		Email: "ana@example.com", PasswordHash: "hash123",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}
	calendars := []*workschedule.WorkCalendar{
		{Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true},
		{Id: 2, StaffId: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true},
		{Id: 3, StaffId: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},
	}
	for _, c := range calendars {
		if err := db.Create(c); err != nil {
			t.Fatalf("seed work calendar: %v", err)
		}
	}

	resp, err := m.GetWorkSchedule(1)
	if err != nil {
		t.Fatalf("GetWorkSchedule: %v", err)
	}
	if resp.StaffName != "Dra. Ana González" {
		t.Errorf("expected staff_name %q, got %q", "Dra. Ana González", resp.StaffName)
	}
	if resp.StaffRole != "Médico General" {
		t.Errorf("expected staff_role %q, got %q", "Médico General", resp.StaffRole)
	}
	if len(resp.Schedule) != 3 {
		t.Fatalf("expected 3 schedule entries, got %d", len(resp.Schedule))
	}
	expectedNames := []string{"Monday", "Wednesday", "Friday"}
	for i, e := range resp.Schedule {
		if e.DayName != expectedNames[i] {
			t.Errorf("entry %d: expected day name %q, got %q", i, expectedNames[i], e.DayName)
		}
		if e.IsActive {
			if e.Start == "" || e.End == "" {
				t.Errorf("entry %d: expected active entry to have start/end times", i)
			}
		} else if e.Start != "" || e.End != "" {
			t.Errorf("entry %d: expected inactive entry to omit start/end times", i)
		}
	}
}

func TestGetWorkSchedule_StaffNotFound(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General", Email: "ana@example.com",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	if _, err := m.GetWorkSchedule(99); err == nil || model.KindOf(err) != model.ErrorNotFound {
		t.Fatalf("expected ErrStaffNotFound, got %v", err)
	}
}

func TestErrStaffNotFoundMessage(t *testing.T) {
	if workschedule.ErrStaffNotFound.Error() != "staff not found" {
		t.Errorf("error message changed: got %q, expected %q", workschedule.ErrStaffNotFound.Error(), "staff not found")
	}
}

// call runs one operation of the module as u1 through the same loopback the demo uses and answers the callback error.
func call(m *workschedule.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(nil, "u1", m).Call(
		router.Operation{Module: workschedule.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}

func seededModule(t *testing.T) *workschedule.Module {
	t.Helper()
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if err := db.Create(&workschedule.Staff{Id: 1, Name: "Dra. Ana González", Role: "Médico General", Email: "ana@example.com"}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}
	if err := db.Create(&workschedule.WorkCalendar{Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true}); err != nil {
		t.Fatalf("seed work calendar: %v", err)
	}
	return m
}

func TestWorkSchedule_ReadByStaff(t *testing.T) {
	m := seededModule(t)
	var got workschedule.WorkScheduleList
	err := call(m, "work_schedule", model.Read, &crud.ReadArgs{Filter: &workschedule.WorkSchedule{StaffId: 1}}, &got)
	if err != nil {
		t.Fatalf("read work_schedule: %v", err)
	}
	if len(got) != 1 || got[0].StaffName != "Dra. Ana González" || got[0].StaffRole != "Médico General" {
		t.Fatalf("expected one schedule of Dra. Ana González, got %+v", got)
	}
	if len(got[0].Schedule) != 1 || got[0].Schedule[0].DayName != "Monday" || got[0].Schedule[0].Start != "09:00" {
		t.Errorf("unexpected entries: %+v", got[0].Schedule)
	}
}

func TestWorkSchedule_NoFilterIsError(t *testing.T) {
	m := seededModule(t)
	var got workschedule.WorkScheduleList
	err := call(m, "work_schedule", model.Read, &crud.ReadArgs{Filter: &workschedule.WorkSchedule{}}, &got)
	if failedField(err) != workschedule.FieldStaffId {
		t.Errorf("expected a refusal on %s, got %v (field %q)", workschedule.FieldStaffId, err, failedField(err))
	}
	if err := call(m, "work_schedule", model.Read, &crud.ReadArgs{}, &got); err == nil {
		t.Error("a read without filter must be refused")
	}
}

func TestWorkSchedule_UnknownStaffIsNotFound(t *testing.T) {
	m := seededModule(t)
	var got workschedule.WorkScheduleList
	err := call(m, "work_schedule", model.Read, &crud.ReadArgs{Filter: &workschedule.WorkSchedule{StaffId: 99}}, &got)
	if model.KindOf(err) != model.ErrorNotFound {
		t.Errorf("expected not found, got %v", err)
	}
}

func TestWorkSchedule_NoWriteOperations(t *testing.T) {
	m := seededModule(t)
	err := call(m, "work_schedule", model.Create, &crud.CreateArgs{Records: []model.Model{&workschedule.WorkSchedule{StaffName: "x"}}}, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("expected an unknown operation, got %v", err)
	}
}

func TestMount_MatchesDefinition(t *testing.T) {
	m := seededModule(t)
	if err := crud.Mount(nil, nil, nil, m); err != nil {
		t.Errorf("Mount: %v", err)
	}
}

func TestItemsDeclaredOnFields(t *testing.T) {
	rec := workschedule.ScheduleEntry{
		Day:      1,
		DayName:  "Monday",
		IsActive: true,
		Start:    "08:00",
		End:      "12:00",
	}
	item := rec.Item()
	if item.Label != "Monday" {
		t.Errorf("expected Label 'Monday', got %q", item.Label)
	}
	if item.Description != "true · 08:00 · 12:00" {
		t.Errorf("expected Description 'true · 08:00 · 12:00', got %q", item.Description)
	}
	if item.ID != "1" {
		t.Errorf("expected ID '1', got %q", item.ID)
	}

	rec2 := workschedule.ScheduleEntry{
		Day:      5,
		DayName:  "Friday",
		IsActive: false,
	}
	item2 := rec2.Item()
	if item2.Description != "false" {
		t.Errorf("expected Description 'false', got %q", item2.Description)
	}
}
