package workschedule

import (
	"strings"
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sqlite"
)

func setupTestModule(t *testing.T) *Module {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	db.CreateTable(&Staff{})
	db.CreateTable(&WorkCalendar{})
	return New(db)
}

func TestGetWorkSchedule_ValidStaff(t *testing.T) {
	m := setupTestModule(t)

	// Seed Staff
	err := m.db.Create(&Staff{
		ID:           1,
		Name:         "Dra. Ana González",
		Role:         "Médico General",
		Email:        "ana@example.com",
		PasswordHash: "hash123",
	})
	if err != nil {
		t.Fatalf("failed to seed staff: %v", err)
	}

	// Seed WorkCalendar
	calendars := []*WorkCalendar{
		{ID: 1, StaffID: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true}, // Monday
		{ID: 2, StaffID: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true}, // Wednesday
		{ID: 3, StaffID: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},          // Friday
	}
	for _, c := range calendars {
		err := m.db.Create(c)
		if err != nil {
			t.Fatalf("failed to seed work calendar: %v", err)
		}
	}

	args := &getWorkScheduleArgs{StaffID: 1}
	var b string
	json.Encode(args, &b)
	res, err := m.GetWorkSchedule(context.Background(), mcp.Request{
		Params: mcp.CallToolParams{Arguments: b},
		Action: 'r',
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.IsError {
		t.Fatalf("expected no error in result, got content: %s", res.Content)
	}

	var envelope mcp.TextContent
	content := res.Content
	if err := json.Decode(res.Content, &envelope); err == nil && envelope.Type == "text" {
		content = envelope.Text
	}

	var staffRes staffResponse
	if err := json.Decode(content, &staffRes); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if staffRes.StaffName != "Dra. Ana González" {
		t.Errorf("expected staff_name 'Dra. Ana González', got %q", staffRes.StaffName)
	}
	if staffRes.StaffRole != "Médico General" {
		t.Errorf("expected staff_role 'Médico General', got %q", staffRes.StaffRole)
	}

	if len(staffRes.Schedule) != 3 {
		t.Fatalf("expected 3 schedule entries, got %d", len(staffRes.Schedule))
	}

	expectedNames := []string{"Lunes", "Miércoles", "Viernes"}
	for i, e := range staffRes.Schedule {
		if e.DayName != expectedNames[i] {
			t.Errorf("expected day name %q, got %q", expectedNames[i], e.DayName)
		}
		if e.IsActive {
			if e.Start == "" || e.End == "" {
				t.Errorf("expected active entry to have start/end times")
			}
		} else {
			if e.Start != "" || e.End != "" {
				t.Errorf("expected inactive entry to omit start/end times")
			}
		}
	}
}

func TestGetWorkSchedule_StaffNotFound(t *testing.T) {
	m := setupTestModule(t)

	// Seed staffID 1
	err := m.db.Create(&Staff{
		ID:           1,
		Name:         "Dra. Ana González",
		Role:         "Médico General",
		Email:        "ana@example.com",
		PasswordHash: "hash123",
	})
	if err != nil {
		t.Fatalf("failed to seed staff: %v", err)
	}

	// Call for staffID 99
	args := &getWorkScheduleArgs{StaffID: 99}
	var b string
	json.Encode(args, &b)
	res, err := m.GetWorkSchedule(context.Background(), mcp.Request{
		Params: mcp.CallToolParams{Arguments: b},
		Action: 'r',
	})

	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error in result, got success")
	}
	if !strings.Contains(res.Content, "staff not found") {
		t.Errorf("expected error containing 'staff not found', got %q", res.Content)
	}
}

func TestGetWorkSchedule_MissingParam(t *testing.T) {
	m := setupTestModule(t)

	args := &getWorkScheduleArgs{} // empty staffID (0)
	var b string
	json.Encode(args, &b)
	res, err := m.GetWorkSchedule(context.Background(), mcp.Request{
		Params: mcp.CallToolParams{Arguments: b},
		Action: 'r',
	})

	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error in result, got success")
	}
	if !strings.Contains(res.Content, "invalid params") && !strings.Contains(res.Content, "params invalid") {
		t.Errorf("expected error containing 'invalid params' or 'params invalid', got %q", res.Content)
	}
}

func TestGetWorkSchedule_InvalidParam(t *testing.T) {
	m := setupTestModule(t)

	// To simulate invalid param with tinywasm/json, we can either pass invalid JSON string
	// or empty arguments if the Bind expects specific fields.
	res, err := m.GetWorkSchedule(context.Background(), mcp.Request{
		Params: mcp.CallToolParams{Arguments: `{"staff_id": "not-a-number"}`},
		Action: 'r',
	})

	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error in result, got success")
	}
	if !strings.Contains(res.Content, "invalid params") && !strings.Contains(res.Content, "params invalid") {
		t.Errorf("expected error containing 'invalid params' or 'params invalid', got %q", res.Content)
	}
}

func TestGetWorkSchedule_DBFailure(t *testing.T) {
	m := setupTestModule(t)

	// Drop table staff before call to simulate DB failure
	m.db.DropTable(&Staff{})

	args := &getWorkScheduleArgs{StaffID: 1}
	var b string
	json.Encode(args, &b)
	res, err := m.GetWorkSchedule(context.Background(), mcp.Request{
		Params: mcp.CallToolParams{Arguments: b},
		Action: 'r',
	})
	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error result due to DB failure, got success")
	}
}
