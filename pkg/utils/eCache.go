package utils

import (
	"container/list"
	"sync"
	"time"
)

type timeKey struct {
	inTime time.Time
	key    string
}

type ECache struct {
	keyQueue *list.List
	repeats  map[string]int
	datas    map[string]int
	mu       sync.Mutex
	ttl      time.Duration
}

func NewECache(ttl time.Duration) *ECache {
	return &ECache{
		datas:    make(map[string]int),
		repeats:  make(map[string]int),
		keyQueue: list.New(),
		ttl:      ttl,
	}
}

func (ec *ECache) Set(key string, data int) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.refresh()

	_, exist := ec.datas[key]
	ec.datas[key] = data
	ec.keyQueue.PushBack(timeKey{inTime: time.Now(), key: key})

	if exist {
		ec.repeats[key]++
	}
}

func (ec *ECache) Get(key string) (int, bool) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.refresh()
	d, ok := ec.datas[key]
	return d, ok
}

func (ec *ECache) Len() int {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.refresh()
	return len(ec.datas)
}

func (ec *ECache) refresh() {
	if ec.ttl <= 0 || ec.keyQueue.Len() == 0 {
		return
	}
	for {
		item := ec.keyQueue.Front()
		if item == nil {
			return
		}
		tKey := item.Value.(timeKey)
		if time.Since(tKey.inTime) <= ec.ttl {
			break
		}
		ec.keyQueue.Remove(item)
		if val, ok := ec.repeats[tKey.key]; !ok {
			delete(ec.datas, tKey.key)
		} else if val <= 1 {
			delete(ec.repeats, tKey.key)
			delete(ec.datas, tKey.key)
		} else {
			ec.repeats[tKey.key] = val - 1
		}
	}
}
