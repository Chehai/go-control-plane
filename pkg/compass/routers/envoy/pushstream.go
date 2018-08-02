package envoy

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

type pushStream struct {
	grpcStream
	sync.Mutex
}

type pushStreams struct {
	Streams map[string][]*pushStream
	sync.RWMutex
}

func (ps *pushStreams) init(keys []string) {
	ps.Streams = make(map[string][]*pushStream)
	for _, k := range keys {
		ps.Streams[k] = make([]*pushStream, 0)
	}
}

func (ps *pushStreams) get(key string) []*pushStream {
	ps.RLock()
	defer ps.RUnlock()
	return ps.Streams[key]
}

func (ps *pushStreams) create(key string, s grpcStream) error {
	log.Debugf("pushtreams:create %v %v\n", key, s)
	strm := pushStream{grpcStream: s}
	ps.Lock()
	defer ps.Unlock()
	strms, ok := ps.Streams[key]
	if !ok {
		return fmt.Errorf("Cannot find push streams for %s", key)
	}
	strms = append(strms, &strm)
	ps.Streams[key] = strms
	return nil
}

func (ps *pushStreams) delete(key string, s grpcStream) error {
	log.Debugf("pushtreams:delete %v %v\n", key, s)
	ps.Lock()
	defer ps.Unlock()
	strms, ok := ps.Streams[key]
	if !ok {
		return fmt.Errorf("Cannot find push streams for %s", key)
	}
	for i, strm := range strms {
		if strm.grpcStream == s {
			copy(strms[i:], strms[i+1:])
			strms[len(strms)-1] = nil
			ps.Streams[key] = strms[:len(strms)-1]
			return nil
		}
	}
	return fmt.Errorf("Cannot find push stream %v for %s", s, key)
}

func (ps *pushStreams) find(s grpcStream) *pushStream {
	ps.RLock()
	defer ps.RUnlock()
	for _, strms := range ps.Streams {
		for _, strm := range strms {
			if strm.grpcStream == s {
				return strm
			}
		}
	}
	return nil
}
