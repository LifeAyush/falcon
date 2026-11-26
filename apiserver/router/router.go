package router

import (
	"net/http/pprof"

	"github.com/gorilla/mux"

	"github.com/LifeAyush/falcon/apiserver/controllers"
	"github.com/LifeAyush/falcon/pkg/parsing"
)

func InitRouter(router *mux.Router) {
	if router == nil {
		panic("mux.Router is nil")
	}

	router.HandleFunc("/apiStatus/backendChange", controllers.BackendChange)
	if parsing.GetConfig().Core.EnableServerStats {
		router.HandleFunc("/apiStatus/reqCount", controllers.StatusReqCount)
	}
	if parsing.GetConfig().Core.EnablePprof {
		// Opt-in profiling, served on the management port only.
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
		router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
	}
}
