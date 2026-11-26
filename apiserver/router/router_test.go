package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	"github.com/LifeAyush/falcon/pkg/parsing"
)

func loadCore(t *testing.T, core string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	content := "[core]\nlisten=18000\napiport=19900\n" + core + "\n[proxy]\n[[proxy.routes]]\nendpoint=\"/\"\nmatch=1\n[[proxy.routes.backends]]\nurl=\"http://127.0.0.1:9\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	parsing.ResetForTest()
	parsing.ConfigFile.FilePath = path
	_ = parsing.GetConfig()
}

func status(t *testing.T, path string) int {
	t.Helper()
	m := mux.NewRouter()
	InitRouter(m)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

func TestPprofOffByDefault(t *testing.T) {
	loadCore(t, "")
	if parsing.GetConfig().Core.EnablePprof {
		t.Fatal("enablePprof must default to false")
	}
	for _, p := range []string{"/debug/pprof/", "/debug/pprof/heap", "/debug/pprof/cmdline"} {
		if got := status(t, p); got != http.StatusNotFound {
			t.Errorf("%s: got %d, want 404 when pprof is disabled", p, got)
		}
	}
}

func TestPprofEnabled(t *testing.T) {
	loadCore(t, "enablePprof=true")
	for _, p := range []string{"/debug/pprof/", "/debug/pprof/heap", "/debug/pprof/cmdline"} {
		if got := status(t, p); got != http.StatusOK {
			t.Errorf("%s: got %d, want 200 when pprof is enabled", p, got)
		}
	}
}
