package framework

import (
	"github.com/dianpeng/moons/http/runtime"
)

type ServiceContext interface {
	Runtime() *runtime.Runtime
	HplSessionWrapper() runtime.SessionWrapper
}
