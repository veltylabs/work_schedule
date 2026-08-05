package tests

import (
	"testing"

	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router/mock"
	"github.com/tinywasm/storage/mem"
	workschedule "github.com/veltylabs/work_schedule"
)

// Implementación de dobles de prueba para ejercitar los métodos EncodeFields y DecodeFields de ormc.
type dummyWriter struct{}

func (w dummyWriter) String(name, val string)                  {}
func (w dummyWriter) Int(name string, val int64)                {}
func (w dummyWriter) Float(name string, val float64)            {}
func (w dummyWriter) Bool(name string, val bool)                {}
func (w dummyWriter) Bytes(name string, val []byte)             {}
func (w dummyWriter) Null(name string)                          {}
func (w dummyWriter) Raw(name, val string)                      {}
func (w dummyWriter) Object(name string, val model.Encodable)   {}
func (w dummyWriter) Array(name string, n int) model.ArrayWriter { return dummyArrayWriter{} }

type dummyArrayWriter struct{}

func (a dummyArrayWriter) String(val string)        {}
func (a dummyArrayWriter) Int(val int64)            {}
func (a dummyArrayWriter) Float(val float64)        {}
func (a dummyArrayWriter) Bool(val bool)            {}
func (a dummyArrayWriter) Bytes(val []byte)         {}
func (a dummyArrayWriter) Object(val model.Encodable) {}
func (a dummyArrayWriter) Close()                   {}

type dummyReader struct{}

func (r dummyReader) String(name string) (string, bool)      { return "", true }
func (r dummyReader) Int(name string) (int64, bool)          { return 0, true }
func (r dummyReader) Float(name string) (float64, bool)      { return 0.0, true }
func (r dummyReader) Bool(name string) (bool, bool)          { return false, true }
func (r dummyReader) Bytes(name string) ([]byte, bool)       { return nil, true }
func (r dummyReader) Object(name string, into model.Decodable) bool { return true }
func (r dummyReader) Array(name string) (model.ArrayReader, bool) { return dummyArrayReader{}, true }
func (r dummyReader) Raw(name string) (string, bool)         { return "", true }

type dummyArrayReader struct{}

func (a dummyArrayReader) Len() int                       { return 1 }
func (a dummyArrayReader) String(i int) string             { return "" }
func (a dummyArrayReader) Int(i int) int64                 { return 0 }
func (a dummyArrayReader) Float(i int) float64             { return 0 }
func (a dummyArrayReader) Bool(i int) bool                 { return false }
func (a dummyArrayReader) Bytes(i int) []byte              { return nil }
func (a dummyArrayReader) Object(i int, into model.Decodable) bool { return true }

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
		{Id: 1, StaffId: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true}, // Lunes
		{Id: 2, StaffId: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true}, // Miércoles
		{Id: 3, StaffId: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},          // Viernes
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

	// Sin autorización configurada: el valor cero de mock.Config deniega cada ruta protegida.
	denied := &mock.Context{}
	denied.SetUserID("u1")
	denied.InBody = []byte(`{"staff_id":1}`)
	reg.Invoke("OP", "/get_work_schedule", denied)
	if denied.Status != 403 {
		t.Fatalf("expected 403 with no Authorize configured, got %d", denied.Status)
	}

	// Configurar una autorización que permita todo, luego reintentar.
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})
	ok := &mock.Context{}
	ok.SetUserID("u1")
	ok.InBody = []byte(`{"staff_id":1}`)
	reg.Invoke("OP", "/get_work_schedule", ok)
	if ok.Status != 0 { // el manejador nunca llama a WriteStatus en caso de éxito
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

func TestModuleModelName(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	if m.ModelName() != "work_schedule" {
		t.Errorf("expected work_schedule, got %q", m.ModelName())
	}
}

func TestAdditionalORMQueries(t *testing.T) {
	db := orm.New(mem.New())
	// Let's seed Staff
	if err := db.Create(&workschedule.Staff{
		Id: 1, Name: "Dr. House", Role: "Diagnostic", Email: "house@clinic.com",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// ReadAllStaff
	staffList, err := workschedule.ReadAllStaff(db.Query(&workschedule.Staff{}))
	if err != nil {
		t.Fatalf("ReadAllStaff: %v", err)
	}
	if len(staffList) != 1 {
		t.Errorf("expected 1 staff, got %d", len(staffList))
	}

	// Let's seed WorkCalendar
	if err := db.Create(&workschedule.WorkCalendar{
		Id: 10, StaffId: 1, DayOfWeek: 2, StartTime: "08:00", EndTime: "12:00", IsActive: true,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// ReadOneWorkCalendar
	cal := &workschedule.WorkCalendar{}
	cal, err = workschedule.ReadOneWorkCalendar(
		db.Query(cal).Where(workschedule.WorkCalendar_.Id).Eq(10),
		cal,
	)
	if err != nil {
		t.Fatalf("ReadOneWorkCalendar: %v", err)
	}
	if cal == nil || cal.Id != 10 {
		t.Errorf("expected calendar id 10, got %v", cal)
	}
}

func TestORMGeneratedCodeAndMetadata(t *testing.T) {
	dw := dummyWriter{}
	dr := dummyReader{}

	// 1. Staff
	s := &workschedule.Staff{Id: 1, Name: "A", Role: "B", Email: "c", PasswordHash: "d"}
	if s.ModelName() != "staff" {
		t.Errorf("Staff.ModelName error")
	}
	if len(s.Schema()) == 0 {
		t.Errorf("Staff.Schema error")
	}
	if len(s.Pointers()) != 4 {
		t.Errorf("Staff.Pointers error")
	}
	if s.IsNil() {
		t.Errorf("Staff.IsNil error")
	}
	s.EncodeFields(dw)
	if err := s.Validate(0); err != nil {
		t.Errorf("Staff.Validate error: %v", err)
	}
	s.DecodeFields(dr)

	var sl workschedule.StaffList
	if sl.Schema() != nil {
		t.Errorf("StaffList.Schema error")
	}
	if sl.Pointers() != nil {
		t.Errorf("StaffList.Pointers error")
	}
	if sl.Len() != 0 {
		t.Errorf("StaffList.Len error")
	}
	if sl.IsNil() {
		t.Errorf("StaffList.IsNil error")
	}
	if !((*workschedule.StaffList)(nil)).IsNil() {
		t.Errorf("StaffList IsNil nil pointer error")
	}
	sl.EncodeFields(dw)
	sl.DecodeFields(dr)
	elem := sl.Append()
	if sl.Len() != 1 {
		t.Errorf("StaffList.Len error after append")
	}
	if sl.At(0) != elem {
		t.Errorf("StaffList.At error")
	}

	// 2. WorkCalendar
	wc := &workschedule.WorkCalendar{Id: 1, StaffId: 2, DayOfWeek: 3, StartTime: "a", EndTime: "b", IsActive: true}
	if wc.ModelName() != "workcalendar" {
		t.Errorf("WorkCalendar.ModelName error")
	}
	if len(wc.Schema()) == 0 {
		t.Errorf("WorkCalendar.Schema error")
	}
	if len(wc.Pointers()) != 6 {
		t.Errorf("WorkCalendar.Pointers error")
	}
	if wc.IsNil() {
		t.Errorf("WorkCalendar.IsNil error")
	}
	wc.EncodeFields(dw)
	if err := wc.Validate(0); err != nil {
		t.Errorf("WorkCalendar.Validate error: %v", err)
	}
	wc.DecodeFields(dr)

	var wcl workschedule.WorkCalendarList
	if wcl.Schema() != nil {
		t.Errorf("WorkCalendarList.Schema error")
	}
	if wcl.Pointers() != nil {
		t.Errorf("WorkCalendarList.Pointers error")
	}
	if wcl.Len() != 0 {
		t.Errorf("WorkCalendarList.Len error")
	}
	if wcl.IsNil() {
		t.Errorf("WorkCalendarList.IsNil error")
	}
	if !((*workschedule.WorkCalendarList)(nil)).IsNil() {
		t.Errorf("WorkCalendarList IsNil nil pointer error")
	}
	wcl.EncodeFields(dw)
	wcl.DecodeFields(dr)
	elemWc := wcl.Append()
	if wcl.Len() != 1 {
		t.Errorf("WorkCalendarList.Len error after append")
	}
	if wcl.At(0) != elemWc {
		t.Errorf("WorkCalendarList.At error")
	}

	// 3. GetWorkScheduleArgs
	gwsa := &workschedule.GetWorkScheduleArgs{StaffId: 1}
	if gwsa.ModelName() != "get_work_schedule_args" {
		t.Errorf("GetWorkScheduleArgs.ModelName error")
	}
	if len(gwsa.Schema()) == 0 {
		t.Errorf("GetWorkScheduleArgs.Schema error")
	}
	if len(gwsa.Pointers()) != 1 {
		t.Errorf("GetWorkScheduleArgs.Pointers error")
	}
	if gwsa.IsNil() {
		t.Errorf("GetWorkScheduleArgs.IsNil error")
	}
	gwsa.EncodeFields(dw)
	if err := gwsa.Validate(0); err != nil {
		t.Errorf("GetWorkScheduleArgs.Validate error: %v", err)
	}
	gwsa.DecodeFields(dr)

	var gwsal workschedule.GetWorkScheduleArgsList
	if gwsal.Schema() != nil {
		t.Errorf("GetWorkScheduleArgsList.Schema error")
	}
	if gwsal.Pointers() != nil {
		t.Errorf("GetWorkScheduleArgsList.Pointers error")
	}
	if gwsal.Len() != 0 {
		t.Errorf("GetWorkScheduleArgsList.Len error")
	}
	if gwsal.IsNil() {
		t.Errorf("GetWorkScheduleArgsList.IsNil error")
	}
	if !((*workschedule.GetWorkScheduleArgsList)(nil)).IsNil() {
		t.Errorf("GetWorkScheduleArgsList IsNil nil pointer error")
	}
	gwsal.EncodeFields(dw)
	gwsal.DecodeFields(dr)
	elemGwsa := gwsal.Append()
	if gwsal.Len() != 1 {
		t.Errorf("GetWorkScheduleArgsList.Len error after append")
	}
	if gwsal.At(0) != elemGwsa {
		t.Errorf("GetWorkScheduleArgsList.At error")
	}

	// 4. ScheduleEntry
	se := &workschedule.ScheduleEntry{Day: 1, DayName: "A", IsActive: true, Start: "B", End: "C"}
	if se.ModelName() != "schedule_entry" {
		t.Errorf("ScheduleEntry.ModelName error")
	}
	if len(se.Schema()) == 0 {
		t.Errorf("ScheduleEntry.Schema error")
	}
	if len(se.Pointers()) != 5 {
		t.Errorf("ScheduleEntry.Pointers error")
	}
	if se.IsNil() {
		t.Errorf("ScheduleEntry.IsNil error")
	}
	se.EncodeFields(dw)
	if err := se.Validate(0); err != nil {
		t.Errorf("ScheduleEntry.Validate error: %v", err)
	}
	se.DecodeFields(dr)

	var sel workschedule.ScheduleEntryList
	if sel.Schema() != nil {
		t.Errorf("ScheduleEntryList.Schema error")
	}
	if sel.Pointers() != nil {
		t.Errorf("ScheduleEntryList.Pointers error")
	}
	if sel.Len() != 0 {
		t.Errorf("ScheduleEntryList.Len error")
	}
	if sel.IsNil() {
		t.Errorf("ScheduleEntryList.IsNil error")
	}
	if !((*workschedule.ScheduleEntryList)(nil)).IsNil() {
		t.Errorf("ScheduleEntryList IsNil nil pointer error")
	}
	sel.EncodeFields(dw)
	sel.DecodeFields(dr)
	elemSe := sel.Append()
	if sel.Len() != 1 {
		t.Errorf("ScheduleEntryList.Len error after append")
	}
	if sel.At(0) != elemSe {
		t.Errorf("ScheduleEntryList.At error")
	}

	// 5. StaffResponse
	sr := &workschedule.StaffResponse{StaffName: "A", StaffRole: "B", Schedule: []workschedule.ScheduleEntry{}}
	if sr.ModelName() != "staff_response" {
		t.Errorf("StaffResponse.ModelName error")
	}
	if len(sr.Schema()) == 0 {
		t.Errorf("StaffResponse.Schema error")
	}
	if len(sr.Pointers()) != 3 {
		t.Errorf("StaffResponse.Pointers error")
	}
	if sr.IsNil() {
		t.Errorf("StaffResponse.IsNil error")
	}
	sr.EncodeFields(dw)
	if err := sr.Validate(0); err != nil {
		t.Errorf("StaffResponse.Validate error: %v", err)
	}
	sr.DecodeFields(dr)

	var srl workschedule.StaffResponseList
	if srl.Schema() != nil {
		t.Errorf("StaffResponseList.Schema error")
	}
	if srl.Pointers() != nil {
		t.Errorf("StaffResponseList.Pointers error")
	}
	if srl.Len() != 0 {
		t.Errorf("StaffResponseList.Len error")
	}
	if srl.IsNil() {
		t.Errorf("StaffResponseList.IsNil error")
	}
	if !((*workschedule.StaffResponseList)(nil)).IsNil() {
		t.Errorf("StaffResponseList IsNil nil pointer error")
	}
	srl.EncodeFields(dw)
	srl.DecodeFields(dr)
	elemSr := srl.Append()
	if srl.Len() != 1 {
		t.Errorf("StaffResponseList.Len error after append")
	}
	if srl.At(0) != elemSr {
		t.Errorf("StaffResponseList.At error")
	}
}
