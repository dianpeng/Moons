package framework

import (
	"github.com/dianpeng/moons/hrouter"
	"github.com/dianpeng/moons/pl"
	"net/http"
)

// when a session finish its execution, it returns back a ApplicationResult object
// for exposing back to the hpl environment
type ApplicationResult struct {
	Event   string
	Context pl.Val
}

// entry for handling a single http request/response
type Application interface {
	// Prepare a transparent request object to be used by the session's Accept
	Prepare(*http.Request, hrouter.Params) (interface{}, error)

	// Invoked when the http request is been accepted by the session handler
	Accept(interface{}, ServiceContext) (ApplicationResult, error)

	// The service session is terminated
	Done(interface{})
}

type ApplicationFactory interface {
	Create([]pl.Val) (Application, error)
	Name() string
	Comment() string
}

var applicationmap map[string]ApplicationFactory = make(map[string]ApplicationFactory)

func AddApplicationFactory(name string, f ApplicationFactory) {
	applicationmap[name] = f
}

func GetApplicationFactory(name string) ApplicationFactory {
	v, ok := applicationmap[name]
	if ok {
		return v
	} else {
		return nil
	}
}

func NewApplicationResult(event string) ApplicationResult {
	return ApplicationResult{
		Event:   event,
		Context: pl.NewValNull(),
	}
}

func (a *ApplicationResult) AddContext(
	key string,
	value pl.Val,
) {
	if !a.Context.IsMap() {
		a.Context = pl.NewValMap()
	}
	a.Context.AddMap(key, value)
}
