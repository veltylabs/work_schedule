package tests

import (
	"errors"
	"strings"
	"testing"
	"webtyp.com/router"
	"webtyp.com/router/mock"
	"webtyp.com/model"
	workschedule "github.com/veltylabs/work_schedule"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/rpc"
)

func TestErrorsMapping(t *testing.T) {
	db := orm.New(mem.New())
	m := workschedule.New(db)
	reg := &mock.Router{}
	m.MountOperations(reg)
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})

	ctx400 := &mock.Context{}
    ctx400.SetUserID("u1")
	ctx400.InBody = []byte(`invalid json`)
	reg.Invoke("OP", "/get_work_schedule", ctx400)
	if ctx400.Status != 400 {
		t.Errorf("expected 400 for ErrCannotRead, got %d", ctx400.Status)
	}

	ctx404 := &mock.Context{}
    ctx404.SetUserID("u1")
	ctx404.InBody = []byte(`{"staff_id": 99}`)
	reg.Invoke("OP", "/get_work_schedule", ctx404)
	if ctx404.Status != 404 {
		t.Errorf("expected 404 for ErrStaffNotFound, got %d", ctx404.Status)
	}

    reg.Operation("test_500", func(ctx router.Context) {
        rpc.WriteError(ctx, errors.New("db error"))
    })
    ctx500 := &mock.Context{}
    ctx500.SetUserID("u1")
    reg.Invoke("OP", "/test_500", ctx500)
    if ctx500.Status != 500 {
        t.Errorf("expected 500 for internal error, got %d", ctx500.Status)
    }
    if strings.Contains(string(ctx500.ResponseBody()), "db error") {
        t.Errorf("expected no error text in response, got %s", ctx500.ResponseBody())
    }
}
