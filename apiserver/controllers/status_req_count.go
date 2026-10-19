package controllers

import (
	"fmt"
	"net/http"

	"github.com/LifeAyush/falcon/apiserver/common"
	"github.com/LifeAyush/falcon/apiserver/resultcodes"
	"github.com/LifeAyush/falcon/pkg/constraints/status"
)

func StatusReqCount(w http.ResponseWriter, r *http.Request) {
	err := common.Report(w, resultcodes.SUCCESS, "success", common.H{
		"inFlight": status.Instance().GetReqCount(),
		"count":    status.Instance().GetReqCount(), // alias for older clients
	})
	if err != nil {
		fmt.Println(err)
	}
}
