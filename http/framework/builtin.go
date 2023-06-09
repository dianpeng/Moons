package framework

import (
	"fmt"
	"github.com/dianpeng/moons/hpl"
	"github.com/dianpeng/moons/hrouter"
	"github.com/dianpeng/moons/pl"
	"net/http"
)

// builtin middleware
type event struct {
	args []pl.Val
}

func (e *event) Name() string {
	return "event"
}

func (e *event) Accept(
	_ *http.Request,
	_ hrouter.Params,
	w HttpResponseWriter,
	ctx ServiceContext,
) bool {
	cfg := hpl.NewPLConfig(ctx.Runtime().Eval, e.args)
	eventName := ""
	context := pl.NewValNull()

	if err := cfg.GetStr(0, &eventName); err != nil {
		w.ReplyError(
			"event.hpl",
			500,
			err,
		)
		return false
	}
	cfg.TryGet(1, &context, pl.NewValNull())

	// run the event
	if _, err := ctx.Runtime().Emit(eventName, context); err != nil {
		w.ReplyError(
			fmt.Sprintf("event.%s", eventName),
			500,
			err,
		)
		return false
	}

	return true
}

type eventfactory struct{}

func (_ *eventfactory) Create(x []pl.Val) (Middleware, error) {
	return &event{args: x}, nil
}

func (_ *eventfactory) Name() string {
	return "event"
}

func (_ *eventfactory) Comment() string {
	return "emit a specific event and run corresponding PL entry synchronously"
}

// builtin application
type eventApp struct {
	args []pl.Val
}

func (e *eventApp) Prepare(*http.Request, hrouter.Params) (interface{}, error) {
	return nil, nil
}

func (e *eventApp) Accept(_ interface{}, ctx ServiceContext) (ApplicationResult, error) {
	cfg := hpl.NewPLConfig(ctx.Runtime().Eval, e.args)
	eventName := ""
	if err := cfg.GetStr(0, &eventName); err != nil {
		return ApplicationResult{}, err
	}
	o := ApplicationResult{
		Event: eventName,
	}
	cfg.TryGet(1, &o.Context, pl.NewValNull())
	return o, nil
}

func (e *eventApp) Done(interface{}) {
}

type eventappfactory struct{}

func (f *eventappfactory) Create(a []pl.Val) (Application, error) {
	return &eventApp{args: a}, nil
}

func (_ *eventappfactory) Name() string {
	return "event"
}

func (_ *eventappfactory) Comment() string {
	return "emit event when application is triggered"
}

func init() {
	AddResponseFactory(
		"event",
		&eventfactory{},
	)
	AddRequestFactory(
		"event",
		&eventfactory{},
	)
	AddApplicationFactory(
		"event",
		&eventappfactory{},
	)
}
