package constraints

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LifeAyush/falcon/pkg/parsing"
	"github.com/LifeAyush/falcon/pkg/utils"
)

func TestExceededLimitReq(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[core]
listen=18000
apiport=19900
[limitReq]
enable=true
interval=60000
frequency=3
[proxy]
[[proxy.routes]]
endpoint="/"
match=1
[[proxy.routes.backends]]
url="http://127.0.0.1:9"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	parsing.ResetForTest()
	parsing.ConfigFile.FilePath = path
	_ = parsing.GetConfig()

	InitLimitCache(utils.NewECache(time.Minute))
	req := httptest.NewRequest(http.MethodGet, "http://example/x", nil)

	for i := 1; i <= 3; i++ {
		if ExceededLimitReq("1.2.3.4", req) {
			t.Fatalf("should allow request %d", i)
		}
	}
	if !ExceededLimitReq("1.2.3.4", req) {
		t.Fatal("expected limit exceeded on 4th request")
	}
}
