//go:build browser

package seed

import (
	"webtyp.com/view"
	workschedule "github.com/veltylabs/work_schedule"
)

func screenDemo(d view.BrowserDeps) []view.Screen {
	return []view.Screen{
		{
			ID: "work_schedule",
			Label: "Schedule",
			Icon: "schedule",
			Permission: "work_schedule",
			Presenter: workschedule.NewView(d.Caller, 1),
		},
	}
}
