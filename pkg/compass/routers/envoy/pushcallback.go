package envoy

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
)

type pushCallback struct {
	context.Context
	Channel chan error
}

type pushCallbacks struct {
	Callbacks map[string]map[int]*pushCallback
	sync.RWMutex
}

func (pc *pushCallbacks) init() {
	pc.Callbacks = make(map[string]map[int]*pushCallback)
}

func (pc *pushCallbacks) create(key1 string, key2 int, ctx context.Context, ch chan error) error {
	cb := pushCallback{Context: ctx, Channel: ch}
	pc.Lock()
	defer pc.Unlock()
	pcb, ok := pc.Callbacks[key1]
	if !ok {
		pcb = make(map[int]*pushCallback)
	}
	pcb[key2] = &cb
	pc.Callbacks[key1] = pcb
	log.Debugf("pushCallbacks.create %s %d", key1, key2)
	return nil
}

func (pc *pushCallbacks) get(key1 string, key2 int) *pushCallback {
	pc.RLock()
	defer pc.RUnlock()
	log.Debugf("pushCallbacks.get %s %s", key1, key2)
	pcb, ok := pc.Callbacks[key1]
	if !ok {
		return nil
	}
	return pcb[key2]
}

func (pc *pushCallbacks) delete(key1 string) {
	pc.Lock()
	pc.Unlock()
	delete(pc.Callbacks, key1)
}
