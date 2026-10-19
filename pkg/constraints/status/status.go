package status

import (
	"sync"
)

var (
	instance     *Status
	instanceOnce sync.Once
)

type Status struct {
	ReqCount   int
	ReqCountMu sync.RWMutex
}

func ensure() *Status {
	instanceOnce.Do(func() {
		instance = &Status{}
	})
	return instance
}

func Init() error {
	_ = ensure()
	return nil
}

func Instance() *Status {
	return ensure()
}

func (sta *Status) AddReqCount() {
	sta.ReqCountMu.Lock()
	sta.ReqCount++
	sta.ReqCountMu.Unlock()
}

func (sta *Status) SubReqCount() {
	sta.ReqCountMu.Lock()
	if sta.ReqCount > 0 {
		sta.ReqCount--
	}
	sta.ReqCountMu.Unlock()
}

func (sta *Status) GetReqCount() int {
	sta.ReqCountMu.RLock()
	defer sta.ReqCountMu.RUnlock()
	return sta.ReqCount
}
