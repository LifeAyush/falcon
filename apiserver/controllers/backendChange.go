package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/LifeAyush/falcon/apiserver/common"
	"github.com/LifeAyush/falcon/apiserver/resultcodes"
	"github.com/LifeAyush/falcon/pkg/dynamic"
)

type Change struct {
	VarName string `json:"varName"`
	Url     string `json:"url"`
}

func BackendChange(w http.ResponseWriter, r *http.Request) {
	var change Change
	success := false

	body, err1 := io.ReadAll(r.Body)
	if err1 != nil {
		fmt.Println(err1)
	} else if err := json.Unmarshal(body, &change); err != nil {
		fmt.Println(err)
	} else if srv, ok := dynamic.DyServers[change.VarName]; !ok {
		success = false
	} else if parsedURL, err := url.Parse(change.Url); err != nil {
		fmt.Println(err)
	} else {
		srv.Mu.Lock()
		srv.URL = parsedURL
		srv.Mu.Unlock()
		success = true
	}

	if err2 := common.Report(w, resultcodes.SUCCESS, "success", common.H{"success": success}); err2 != nil {
		fmt.Println(err2)
	}
}
