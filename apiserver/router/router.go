package router

import (
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
}
