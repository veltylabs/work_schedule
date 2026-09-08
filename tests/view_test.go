package tests

import (
	"testing"

	workschedule "github.com/veltylabs/work_schedule"
	"webtyp.com/json"
	"webtyp.com/orm"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
)

// TestNewView_ListsScheduleEntries: el Presenter aplana las ScheduleEntry de
// get_work_schedule a filas de lista con su Description correcta.
func TestNewView_ListsScheduleEntries(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	_ = m

	// El caller in-proc (loopback) NO está en este módulo (blacklist: no hay
	// transporte). Se usa mock.Caller con CannedResult para simular la op.
	resp := workschedule.StaffResponse{
		StaffName: "Dra. Ana González",
		StaffRole: "Médico General",
		Schedule: []workschedule.ScheduleEntry{
			{Day: 1, DayName: "Lunes", IsActive: true, Start: "08:00", End: "12:00"},
			{Day: 3, DayName: "Miércoles", IsActive: true, Start: "09:00", End: "11:00"},
			{Day: 5, DayName: "Viernes", IsActive: false},
		},
	}
	var canned []byte
	if err := json.Encode(&resp, &canned); err != nil {
		t.Fatalf("encode seeded response: %v", err)
	}

	caller := &mock.Caller{CannedResult: canned}

	pres := workschedule.NewView(caller, 1)
	if err := pres.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	items := pres.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 schedule entries, got %d", len(items))
	}
	// Los dos primeros activos llevan "HH:MM–HH:MM"; el inactivo "Inactivo".
	if items[0].Description != "08:00–12:00" {
		t.Errorf("entry 0 description = %q, want 08:00–12:00", items[0].Description)
	}
	if items[1].Description != "09:00–11:00" {
		t.Errorf("entry 1 description = %q, want 09:00–11:00", items[1].Description)
	}
	if items[2].Description != "Inactivo" {
		t.Errorf("inactive entry description = %q, want Inactivo", items[2].Description)
	}
}
