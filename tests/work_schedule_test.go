package tests

import (
	"testing"
	"webtyp.com/model"
	"webtyp.com/crud"
	"webtyp.com/orm"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
	workschedule "github.com/veltylabs/work_schedule"
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

func TestMountOperations_RoutesAndEnforcesRBAC(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General", Email: "ana@example.com",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}
	if err := db.Create(&workschedule.WorkCalendar{
		Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true,
	}); err != nil {
		t.Fatalf("seed work calendar: %v", err)
	}

	// For RBAC testing we must use mock.Router because crud.Loopback ActingAs ignores permissions
	reg := &mock.Router{}
	crud.Mount(reg, nil, m)

	denied := &mock.Context{}
	denied.SetUserID("u1")
	denied.InBody = []byte(`{"filter": {"staff_id":1}}`)
	reg.Invoke("OP", "/work_schedule_r", denied)
	if denied.Status != 403 {
		t.Fatalf("expected 403 with no Authorize configured, got %d", denied.Status)
	}

	// Configurar un Authorize que permita todo, y luego reintentar.
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool {
			// work_schedule is the module and model name
			return r == "work_schedule" && a == model.Read
		},
	})

	// Para la prueba feliz sí podemos usar loopback (que ignora roles pero invoca directo) o el mock.
	// Usaremos loopback para tener decoding nativo de la respuesta.
	caller := crud.LoopbackActingAs("u1", m)
	var got workschedule.WorkScheduleList
	var rerr error
	caller.Call("work_schedule.work_schedule_r", &crud.ReadArgs{Filter: &workschedule.GetWorkScheduleArgs{StaffId: 1}}, &got, func(err error) { rerr = err })

	if rerr != nil {
		t.Fatalf("Call work_schedule_r failed: %v", rerr)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 decoded work schedule entry, got %d", len(got))
	}

	ws := got[0]
	if ws.StaffName != "Dra. Ana González" {
		t.Errorf("expected decoded staff_name %q, got %q", "Dra. Ana González", ws.StaffName)
	}
	if len(ws.Schedule) != 1 {
		t.Fatalf("expected 1 decoded schedule entry, got %d", len(ws.Schedule))
	}
	if ws.Schedule[0].DayName != "Monday" || ws.Schedule[0].Start != "09:00" {
		t.Errorf("unexpected decoded schedule entry: %+v", ws.Schedule[0])
	}
}

func TestOpGetWorkSchedule_InvalidParam(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)

	caller := crud.LoopbackActingAs("u1", m)
	var got workschedule.WorkScheduleList
	var rerr error

	// Sin filtro
	caller.Call("work_schedule.work_schedule_r", &crud.ReadArgs{}, &got, func(err error) { rerr = err })

	if rerr == nil {
		t.Fatalf("expected error for a missing/malformed filter, got nil")
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
