package application

import (
	"github.com/dianpeng/moons/hpl"
	"github.com/dianpeng/moons/http/framework"
	"github.com/dianpeng/moons/http/runtime"
	"github.com/dianpeng/moons/pl"
)

type bgApplicationWrapper struct {
	parent      runtime.SessionWrapper
	application framework.Application
}

func (b *bgApplicationWrapper) OnLoadVar(x *pl.Evaluator, name string) (pl.Val, error) {
	return b.parent.OnLoadVar(x, name)
}

func (b *bgApplicationWrapper) OnStoreVar(x *pl.Evaluator, name string, value pl.Val) error {
	return b.parent.OnStoreVar(x, name, value)
}

func (b *bgApplicationWrapper) OnAction(x *pl.Evaluator, name string, val pl.Val) error {
	return b.parent.OnAction(x, name, val)
}

func (b *bgApplicationWrapper) GetHttpClient(url string) (hpl.HttpClient, error) {
	// parent's GetHttpClient is always thread safe
	return b.parent.GetHttpClient(url)
}

func newBgApplicationWrapper(
	parent runtime.SessionWrapper,
	application framework.Application,
) *bgApplicationWrapper {
	return &bgApplicationWrapper{
		parent:      parent,
		application: application,
	}
}
