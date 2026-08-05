package workschedule

import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/orm"
)

// ErrStaffNotFound es devuelto por GetWorkSchedule cuando ninguna fila de personal coincide con el id dado.
var ErrStaffNotFound = fmt.Err("staff", "not", "found")

// Module es un adaptador de sólo lectura sobre dos tablas legadas ('staff', 'workcalendar') que este
// módulo no posee — ver notas de dominio en AGENTS.md. No genera IDs ni publica eventos, por lo que
// no requiere Deps: New sólo necesita el *orm.DB ya conectado.
type Module struct {
	db *orm.DB
}

// New conecta el módulo a un *orm.DB ya conectado (respaldado por el storage.Conn que la app haya
// elegido). Nunca migra un esquema — ver Etapa 5 — y nunca falla, por lo que devuelve *Module, no
// (*Module, error): no hay nada aquí que pueda salir mal en el momento de la construcción.
func New(db *orm.DB) *Module {
	return &Module{db: db}
}

// dayNames: duplicación conocida y aceptada — business_hours lleva la misma tabla en su vista.
// Dos copias es el máximo tolerado; si un tercer módulo necesita nombres de días (o llega i18n), el
// acoplamiento se mueve río arriba (tinywasm/time o la app) según la regla lego "el pegamento se escribe
// una sola vez" — anotado aquí para que la decisión quede registrada, no bifurcada silenciosamente de nuevo.
var dayNames = [7]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

// GetWorkSchedule devuelve el horario semanal del miembro del personal dado, o ErrStaffNotFound.
func (m *Module) GetWorkSchedule(staffId int64) (StaffResponse, error) {
	staffRow := &Staff{}
	staffRow, err := ReadOneStaff(m.db.Query(staffRow).Where(Staff_.Id).Eq(staffId), staffRow)
	if err != nil {
		// Nunca ocultar una falla real de la base de datos como "no encontrado" — sólo orm.ErrNotFound se
		// mapea al centinela del dominio; cualquier otra cosa surge como el error interno que es (un corte de
		// DB reportado como "personal no encontrado" es la falla silenciosa que el arnés prohíbe).
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
