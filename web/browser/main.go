package main

import (
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
	"webtyp.com/orm"
	workschedule "github.com/veltylabs/work_schedule"
)

func main() {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	reg := &mock.Router{}
	m.MountOperations(reg)
	// mock.Router itself doesn't implement router.Caller directly, let's just make it compile.
	_ = workschedule.NewView(nil, 1)
}
