package app

// AI 网关端到端集成测试(真实 PG + 可选真实 OpenAI 兼容服务):
// admin 集中配置 apiKey/apiUrl → 业务接口无密钥调用 chat completions。
// 运行: BOSS_PG_TEST_DSN="host=... dbname=boss ..." \
//       BOSS_AI_TEST_URL=https://openai.bowong.cc/v1 BOSS_AI_TEST_KEY=sk-... BOSS_AI_TEST_MODEL=gpt-5.4-mini \
//       go test ./internal/app/ -run TestE2E_AI -v -count=1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func codeOf(out map[string]any) float64 {
	v, _ := out["code"].(float64)
	return v
}

func TestE2E_AIOpenAI_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Database.DSN = dsn
	a, err := New(ctx, cfg, "../../migrations")
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	defer a.Close()

	// 一次性 sysadmin 账号。
	suffix := time.Now().UnixNano()
	username := "e2e-ai-" + fmt.Sprintf("%d", suffix)
	password := "E2e-pass-123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx,
		`INSERT INTO accounts(username, password_hash, real_name, role_id)
		 SELECT $1, $2, 'AI测试员', id FROM roles WHERE code='sysadmin'`,
		username, string(hash)); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	r := gin.New()
	RegisterRoutes(r, a, auth.NewManager("e2e-secret", time.Hour))
	ts := httptest.NewServer(r)
	defer ts.Close()

	token := loginE2E(t, ts, &e2eSeed{username: username, password: password})
	do := func(method, path, body string) map[string]any {
		t.Helper()
		var rdr io.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, rdr)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	// 未配置时业务接口应报 42200。
	if out := do("POST", "/api/v1/ai/chat/completions", `{"messages":[{"role":"user","content":"hi"}]}`); codeOf(out) != float64(apitypes.CodeInvalidParam) {
		t.Fatalf("unconfigured chat: %v", out)
	}

	apiURL := os.Getenv("BOSS_AI_TEST_URL")
	apiKey := os.Getenv("BOSS_AI_TEST_KEY")
	model := os.Getenv("BOSS_AI_TEST_MODEL")
	if apiURL == "" || apiKey == "" {
		t.Log("BOSS_AI_TEST_URL/KEY 未设置,仅验证未配置分支")
		return
	}

	// admin 集中配置(脱敏落库 biz_params)。
	upd := `{"apiUrl":"` + apiURL + `","apiKey":"` + apiKey + `","model":"` + model + `"}`
	if out := do("PUT", "/api/v1/ai/openai/config", upd); codeOf(out) != 0 {
		t.Fatalf("update config: %v", out)
	}

	// 配置视图:configured=true 且密钥脱敏。
	view := do("GET", "/api/v1/ai/openai/config", "")
	if codeOf(view) != 0 || view["data"].(map[string]any)["configured"] != true {
		t.Fatalf("config view: %v", view)
	}

	// 业务调用:请求方不带任何密钥。
	out := do("POST", "/api/v1/ai/chat/completions", `{"messages":[{"role":"user","content":"reply with exactly: ok"}]}`)
	if codeOf(out) != 0 {
		t.Fatalf("chat completions: %v", out)
	}
	data := out["data"].(map[string]any)
	t.Logf("chat ok: model=%s content=%q usage=%v", data["model"], data["content"], data["usage"])
}
