package constraints

import (
	"net/http"

	"github.com/LifeAyush/falcon/pkg/parsing"
	"github.com/LifeAyush/falcon/pkg/utils"
)

var limitReqCache *utils.ECache

// InitLimitCache wires the shared ephemeral cache used for rate limiting.
func InitLimitCache(c *utils.ECache) {
	limitReqCache = c
}

func getReqCount(key string) int {
	if limitReqCache == nil {
		return 0
	}
	hashkey := utils.Hash(key)
	obj, ok := limitReqCache.Get(hashkey)
	if ok {
		limitReqCache.Set(hashkey, obj+1)
		return obj + 1
	}
	limitReqCache.Set(hashkey, 1)
	return 1
}

// ExceededLimitReq reports whether the client has exceeded the configured frequency.
func ExceededLimitReq(ip string, req *http.Request) bool {
	if limitReqCache == nil {
		return false
	}
	key := ip + req.URL.Path
	count := getReqCount(key)
	freq := parsing.GetConfig().LimitReq.Frequency
	if freq <= 0 {
		return false
	}
	return count > freq
}
