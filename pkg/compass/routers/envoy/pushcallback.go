package envoy

import (
	"context"
	"sync"
)

type pushCallback struct {
	context.Context
	Channel chan error
}

type pushCallbacks struct {
	Callbacks map[string]map[string]*pushCallback
	sync.RWMutex
}

func (pc *pushCallbacks) init() {
	pc.Callbacks = make(map[string]map[string]*pushCallback)
}

func (pc *pushCallbacks) create(key1 string, key2 string, ctx context.Context, ch chan error) error {
	cb := pushCallback{Context: ctx, Channel: ch}
	pc.Lock()
	defer pc.Unlock()
	pcb, ok := pc.Callbacks[key1]
	if !ok {
		pcb = make(map[string]*pushCallback)
	}
	pcb[key2] = &cb
	pc.Callbacks[key1] = pcb
	return nil
}

func (pc *pushCallbacks) get(key1 string, key2 string) *pushCallback {
	pc.RLock()
	defer pc.Unlock()
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
