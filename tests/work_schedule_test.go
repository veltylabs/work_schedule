package tests

import (
	"testing"

	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router/mock"
	"github.com/tinywasm/storage/mem"
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
		{Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true}, // Monday
		{Id: 2, StaffId: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true}, // Wednesday
		{Id: 3, StaffId: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},          // Friday
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
	expectedNames := []string{"Lunes", "Miércoles", "Viernes"}
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

	if _, err := m.GetWorkSchedule(99); err != workschedule.ErrStaffNotFound {
		t.Fatalf("expected ErrStaffNotFound, got %v", err)
	}
}

func TestMountOps_RoutesAndEnforcesRBAC(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dra. Ana González", Role: "Médico General", Email: "ana@example.com",
	}); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	reg := &mock.Router{}
	m.MountOps(reg)

	// No Authorize configured: mock.Config's zero value denies every guarded route.
	denied := &mock.Context{}
	denied.SetUserID("u1")
	denied.InBody = []byte(`{"staff_id":1}`)
	reg.Invoke("OP", "/get_work_schedule", denied)
	if denied.Status != 403 {
		t.Fatalf("expected 403 with no Authorize configured, got %d", denied.Status)
	}

	// Configure an Authorize that allows everything, then retry.
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})
	ok := &mock.Context{}
	ok.SetUserID("u1")
	ok.InBody = []byte(`{"staff_id":1}`)
	reg.Invoke("OP", "/get_work_schedule", ok)
	if ok.Status != 0 { // handler never calls WriteStatus on success
		t.Fatalf("expected no error status, got %d", ok.Status)
	}
	if len(ok.ResponseBody()) == 0 {
		t.Fatalf("expected an encoded response body")
	}
}

func TestOpGetWorkSchedule_InvalidParam(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)

	reg := &mock.Router{}
	m.MountOps(reg)
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})

	ctx := &mock.Context{}
	ctx.SetUserID("u1")
	ctx.InBody = []byte(`not valid json`)
	reg.Invoke("OP", "/get_work_schedule", ctx)
	if ctx.Status != 400 {
		t.Fatalf("expected 400 for a malformed body, got %d", ctx.Status)
	}
}
