package vhost

import (
	"github.com/dianpeng/moons/alog"
	"github.com/dianpeng/moons/g"
	"github.com/dianpeng/moons/hpl"
	"github.com/dianpeng/moons/pl"
	"github.com/dianpeng/moons/redis/runtime"
	ru "github.com/dianpeng/moons/redis/util"
	"github.com/dianpeng/moons/util"

	"github.com/tidwall/redcon"

	"fmt"
	"strings"
	"sync"
)

const (
	eventAccept  = "redis.:accept"
	eventClose   = "redis.:close"
	eventCommand = "redis.*"

	// kept in sync with the redis category
	eventCatBitmap      = "reids.:bitmap"
	eventCatGeneric     = "redis.:generic"
	eventCatGeo         = "redis.:geo"
	eventCatHash        = "redis.:hash"
	eventCatHyperLogLog = "redis.:hyperloglog"
	eventCatList        = "redis.:list"
	eventCatPubSub      = "redis.:pubsub"
	eventCatScript      = "redis.:script"
	eventCatSet         = "redis.:set"
	eventCatSortedSet   = "redis.:sorted_set"
	eventCatStream      = "redis.:stream"
	eventCatString      = "redis.:string"
	eventCatTransaction = "redis.:transaction"
)

type servicePool struct {
	idle    []*serviceHandler
	maxSize int
	sync.Mutex
}

type serviceHandler struct {
	runtime          *runtime.Runtime
	vhost            *VHost
	activeHttpClient []*util.HClient
}

func newServicePool(cacheSize int) servicePool {
	csize := cacheSize
	if csize == 0 {
		csize = g.MaxSessionCacheSize
	}
	return servicePool{
		maxSize: csize,
	}
}

func (s *servicePool) idleSize() int {
	s.Lock()
	defer s.Unlock()
	return len(s.idle)
}

func (s *servicePool) get() *serviceHandler {
	s.Lock()
	defer s.Unlock()
	if len(s.idle) == 0 {
		return nil
	}
	idleSize := len(s.idle)
	last := s.idle[idleSize-1]
	s.idle = s.idle[:idleSize-1]
	return last
}

func (s *servicePool) put(h *serviceHandler) bool {
	s.Lock()
	defer s.Unlock()
	if len(s.idle)+1 >= s.maxSize {
		return false
	}
	s.idle = append(s.idle, h)
	return true
}

func newServiceHandler(vhost *VHost) *serviceHandler {
	h := &serviceHandler{
		runtime: runtime.NewRuntimeWithModule(vhost.Module),
		vhost:   vhost,
	}
	return h
}

func (s *serviceHandler) GetHttpClient(url string) (hpl.HttpClient, error) {
	c, err := s.vhost.clientPool.Get(url)
	if err != nil {
		return nil, err
	}
	s.activeHttpClient = append(s.activeHttpClient, &c)
	return &c, nil
}

func (s *serviceHandler) finish() {
	if s.activeHttpClient != nil {
		for _, c := range s.activeHttpClient {
			s.vhost.clientPool.Put(*c)
		}
		s.activeHttpClient = nil
	}
}

func (s *serviceHandler) err(
	c redcon.Conn,
	event string,
	err error,
) {
	c.WriteError(
		fmt.Sprintf("Runtime(%s) error: %s", event, err.Error()),
	)
}

func (s *serviceHandler) onEmptyCommand(
	_ redcon.Conn,
	_ redcon.Command,
) {
}

func (s *serviceHandler) onEvent(
	conn redcon.Conn,
	cmd redcon.Command,
) {
	log := alog.NewLog(s.vhost.LogFormat)

	defer func() {
		s.vhost.uploadLog(&log, nil)
		s.finish()
	}()

	cmdVal := runtime.NewCommandVal(
		&cmd,
	)

	connVal, connStatus := runtime.NewConnectionVal(
		conn,
	)

	defer func() {
		if connStatus.DidWrite() && !connStatus.DidClose() {
			s.onEmptyCommand(conn, cmd)
		}
	}()

	cmdName := strings.ToUpper(string(cmd.Args[0]))
	cmdCatEvent := fmt.Sprintf("redis.:%s", ru.CommandCategoryName(cmdName))
	cmdEvent := fmt.Sprintf("redis.%s", cmdName)

	var err error

	if err = s.runtime.OnInit(
		connVal,
		s,
		&log,
	); err != nil {
		s.err(
			conn,
			"@init",
			err,
		)
		return
	}

	// 1) highest priority, ie the most specific event trigger
	if s.runtime.Module.HaveEvent(cmdEvent) {
		if _, err = s.runtime.Emit(
			cmdEvent,
			cmdVal,
		); err != nil {
			s.err(
				conn,
				cmdEvent,
				err,
			)
		}
		return
	}

	// 2) lower priority, ie the command category event trigger
	if s.runtime.Module.HaveEvent(cmdCatEvent) {
		if _, err = s.runtime.Emit(
			cmdCatEvent,
			cmdVal,
		); err != nil {
			s.err(
				conn,
				cmdEvent,
				err,
			)
		}
		return
	}

	// 3) lastly, use wildcard event trigger to capture the event
	if _, err = s.runtime.Emit(
		eventCommand,
		cmdVal,
	); err != nil {
		s.err(
			conn,
			cmdEvent,
			err,
		)
		return
	}
}

func (s *serviceHandler) onAccept(
	conn redcon.Conn,
) bool {
	log := alog.NewLog(s.vhost.LogFormat)

	defer func() {
		s.vhost.uploadLog(&log, nil)
		s.finish()
	}()

	connVal, _ := runtime.NewConnectionVal(
		conn,
	)

	var err error

	if err = s.runtime.OnInit(
		connVal,
		s,
		&log,
	); err != nil {
		s.err(
			conn,
			"@init",
			err,
		)
		return false
	}

	if val, err := s.runtime.Emit(
		eventAccept,
		pl.NewValNull(),
	); err != nil {
		s.err(
			conn,
			eventAccept,
			err,
		)
		return false
	} else {
		if val.IsBool() {
			return val.Bool()
		} else {
			return true
		}
	}
}

func (s *serviceHandler) onClose(
	conn redcon.Conn,
	connErr error,
) {
	log := alog.NewLog(s.vhost.LogFormat)

	defer func() {
		s.vhost.uploadLog(&log, nil)
		s.finish()
	}()

	connVal, _ := runtime.NewConnectionVal(
		conn,
	)

	var err error
	ctx := pl.NewValNull()
	if connErr != nil {
		ctx = pl.NewValStr(connErr.Error())
	}

	if err = s.runtime.OnInit(
		connVal,
		s,
		&log,
	); err != nil {
		s.err(
			conn,
			"@init",
			err,
		)
		return
	}

	if _, err := s.runtime.Emit(
		eventClose,
		ctx,
	); err != nil {
		s.err(
			conn,
			eventClose,
			err,
		)
	}
}
