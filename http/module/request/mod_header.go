package request

import (
	"github.com/dianpeng/moons/hrouter"
	"github.com/dianpeng/moons/http/framework"
	"github.com/dianpeng/moons/http/module"
	"github.com/dianpeng/moons/pl"
	"net/http"
)

type headerModFn func(string, []pl.Val, http.Header, framework.ServiceContext) error

type modheader struct {
	args  []pl.Val
	name  string
	modFn headerModFn
}

func (e *modheader) Name() string {
	return e.name
}

func (e *modheader) Accept(
	r *http.Request,
	p hrouter.Params,
	w framework.HttpResponseWriter,
	ctx framework.ServiceContext,
) bool {
	if err := e.modFn(
		e.name,
		e.args,
		r.Header,
		ctx,
	); err != nil {
		w.ReplyError(
			e.name,
			500,
			err,
		)
		return false
	}
	return true
}

type modheaderfactory struct {
	name  string
	modFn headerModFn
}

func (e *modheaderfactory) Create(x []pl.Val) (framework.Middleware, error) {
	return &modheader{
		args:  x,
		name:  e.name,
		modFn: e.modFn,
	}, nil
}

func (e *modheaderfactory) Name() string {
	return e.name
}

func (e *modheaderfactory) Comment() string {
	return "modify request's header"
}

func init() {
	framework.AddRequestFactory(
		"header_add",
		&modheaderfactory{
			name:  "request.header_add",
			modFn: module.HeaderAdd,
		},
	)

	framework.AddRequestFactory(
		"header_set",
		&modheaderfactory{
			name:  "request.header_set",
			modFn: module.HeaderSet,
		},
	)

	framework.AddRequestFactory(
		"header_del",
		&modheaderfactory{
			name:  "request.header_del",
			modFn: module.HeaderDel,
		},
	)
}
