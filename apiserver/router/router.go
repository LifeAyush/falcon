package router

import (
	"github.com/gorilla/mux"
	"github.com/LifeAyush/falcon/apiserver/controllers"
)

func InitRouter(router *mux.Router) {
	if router == nil {
		panic("mux.Router is nil")
	}

	router.HandleFunc("/apiStatus/reqCount", controllers.StatusReqCount)
	router.HandleFunc("/apiStatus/backendChange", controllers.BackendChange)
}
