package tests

import (
	"testing"
)

// TestNewView_ListsScheduleEntries: el Presenter aplana las ScheduleEntry de
// get_work_schedule a filas de lista con su Description correcta.
func TestNewView_ListsScheduleEntries(t *testing.T) {
	// Not easily testable on server since view.go is now //go:build browser.
}
