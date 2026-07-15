//go:build !wasm

package workschedule

// Staff maps to the legacy 'staff' table. READ-ONLY — no DDL allowed.
type Staff struct {
	ID           int64  `db:"pk"`
	Name         string `db:"not_null"`
	Role         string `db:"not_null"`
	Email        string `db:"unique"`
	PasswordHash string `db:"-"` // ormc:exclude
}

func (s *Staff) TableName() string { return "staff" }

// WorkCalendar maps to the legacy 'workcalendar' table. READ-ONLY — no DDL allowed.
// ormc:model workcalendar
// ormc:table workcalendar
type WorkCalendar struct {
	ID        int64  `db:"pk"`
	StaffID   int64  `db:"ref=staff,not_null"`
	DayOfWeek int    `db:"not_null"` // 0=Sunday … 6=Saturday
	StartTime string `db:"not_null"` // "HH:MM"
	EndTime   string `db:"not_null"` // "HH:MM"
	IsActive  bool   `db:"not_null"`
}

func (w *WorkCalendar) TableName() string { return "workcalendar" }

// ormc:formonly
type getWorkScheduleArgs struct {
	StaffID int64 ``
}

// ormc:formonly
type scheduleEntry struct {
	Day      int    ``
	DayName  string ``
	IsActive bool   ``
	Start    string `json:",omitempty"`
	End      string `json:",omitempty"`
}

// ormc:formonly
type staffResponse struct {
	StaffName string          ``
	StaffRole string          ``
	Schedule  []scheduleEntry ``
}
