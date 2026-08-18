package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeKeyService 模拟 apikey.Service。
type fakeKeyService struct {
	lookup map[string]int64 // keyHash → accountID
	touchN int
}

func (f *fakeKeyService) Create(ctx context.Context, accountID, createdBy int64, name string) (*apikey.CreateResult, error) {
	return nil, nil
}
func (f *fakeKeyService) List(ctx context.Context) ([]apikey.APIKey, error) { return nil, nil }
func (f *fakeKeyService) Revoke(ctx context.Context, id int64) error        { return nil }
func (f *fakeKeyService) Lookup(ctx context.Context, keyHash string) (int64, error) {
	if id, ok := f.lookup[keyHash]; ok {
		return id, nil
	}
	return 0, apikey.ErrNotFound
}
func (f *fakeKeyService) Touch(ctx context.Context, keyHash string) {
	f.touchN++
}

// fakeUserService 模拟 user.Service(仅测试 GetProfile)。
type fakeUserService struct {
	profiles map[int64]*user.Profile
}

func (f *fakeUserService) GetProfile(ctx context.Context, accountID int64) (*user.Profile, error) {
	if p, ok := f.profiles[accountID]; ok {
		return p, nil
	}
	return nil, user.ErrNotFound
}
func (f *fakeUserService) Login(ctx context.Context, username, password string) (*user.LoginResult, error) {
	return nil, nil
}
func (f *fakeUserService) ListRoles(ctx context.Context) ([]user.Role, error) {
	return nil, nil
}
func (f *fakeUserService) CreateAccount(ctx context.Context, in user.AccountInput) (int64, error) {
	return 0, nil
}
func (f *fakeUserService) UpdateAccount(ctx context.Context, id int64, in user.AccountInput) error {
	return nil
}
func (f *fakeUserService) CreateDepartment(ctx context.Context, legalEntityID int64, name string) (int64, error) {
	return 0, nil
}
func (f *fakeUserService) UpdateDepartment(ctx context.Context, id, legalEntityID int64, name string) error {
	return nil
}
func (f *fakeUserService) CreatePost(ctx context.Context, deptID int64, code, name string, roles []string) (int64, error) {
	return 0, nil
}
func (f *fakeUserService) UpdatePost(ctx context.Context, id, deptID int64, code, name string, roles []string) error {
	return nil
}
func (f *fakeUserService) ListParams(ctx context.Context) ([]user.Param, error) {
	return nil, nil
}
func (f *fakeUserService) UpdateParam(ctx context.Context, key, value string, updatedBy int64) error {
	return nil
}
func (f *fakeUserService) EnsureSuperAdmin(ctx context.Context, username, password, realName string) (bool, error) {
	return false, nil
}
func (f *fakeUserService) HasPermission(ctx context.Context, accountID int64, permCode string) (bool, error) {
	return true, nil
}
func (f *fakeUserService) HasDataScope(ctx context.Context, accountID int64, owner user.DataScope) (bool, error) {
	return true, nil
}
func (f *fakeUserService) ListAddresses(ctx context.Context, parentID int64) ([]user.Address, error) {
	return nil, nil
}
func (f *fakeUserService) ImportAddresses(ctx context.Context, rows []user.AddressRow) (int, error) {
	return 0, nil
}
func (f *fakeUserService) SetAddressGeo(ctx context.Context, id int64, countryCode, adminCode string) error {
	return nil
}
func (f *fakeUserService) ListUnlinkedRoots(ctx context.Context) ([]user.Address, error) {
	return nil, nil
}
func (f *fakeUserService) CreateAddress(ctx context.Context, parentID int64, label, name, countryCode, adminCode string) (int64, error) {
	return 0, nil
}
func (f *fakeUserService) UpdateAddressName(ctx context.Context, id int64, name string) error {
	return nil
}
func (f *fakeUserService) DeleteAddress(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeUserService) SearchAddresses(ctx context.Context, kw string) ([]user.AddressHit, error) {
	return nil, nil
}
func (f *fakeUserService) ListRegions(ctx context.Context, parentPath string) ([]user.Region, error) {
	return nil, nil
}
func (f *fakeUserService) ListLegalEntities(ctx context.Context) ([]user.LegalEntity, error) {
	return nil, nil
}
func (f *fakeUserService) ListAccounts(ctx context.Context) ([]user.AccountRow, error) {
	return nil, nil
}
func (f *fakeUserService) CreateLegalEntity(ctx context.Context, e user.LegalEntity) (int64, error) {
	return 0, nil
}
func (f *fakeUserService) UpdateLegalEntity(ctx context.Context, id int64, e user.LegalEntity) error {
	return nil
}
func (f *fakeUserService) ListMenuPermMatrix(ctx context.Context) (user.MenuPermMatrix, error) {
	return user.MenuPermMatrix{}, nil
}
func (f *fakeUserService) ListDepartments(ctx context.Context, legalEntityID int64) ([]user.Department, error) {
	return nil, nil
}
func (f *fakeUserService) ListPosts(ctx context.Context, deptID int64) ([]user.Post, error) {
	return nil, nil
}
func (f *fakeUserService) GetDataScope(ctx context.Context, accountID int64) (user.DataScope, error) {
	return user.DataScope{}, nil
}

// helper 构造测试路由。
func newAPIKeyRouter(keys apikey.Service, usr user.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authed := r.Group("/api/v1")
	authed.Use(APIKeyAuth(keys, usr), Authn(nil))
	authed.GET("/auth/me", func(c *gin.Context) {
		claims := c.MustGet(CtxClaims).(*auth.Claims)
		c.JSON(http.StatusOK, gin.H{"aid": claims.AccountID, "usr": claims.Username, "role": claims.RoleCode})
	})
	return r
}

func TestAPIKeyAuthValid(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]int64{sha256Hash("boss_valid_key"): 7}}
	usr := &fakeUserService{profiles: map[int64]*user.Profile{
		7: {AccountID: 7, Username: "alice", RoleCode: "ops", RoleName: "运营"},
	}}
	r := newAPIKeyRouter(keys, usr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-API-Key", "boss_valid_key")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want 200 body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"aid":7`) || !strings.Contains(body, `"usr":"alice"`) || !strings.Contains(body, `"role":"ops"`) {
		t.Fatalf("body=%s", body)
	}
	if keys.touchN != 1 {
		t.Fatalf("touchN=%d want 1", keys.touchN)
	}
}

func TestAPIKeyAuthInvalid(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]int64{}}
	usr := &fakeUserService{profiles: map[int64]*user.Profile{}}
	r := newAPIKeyRouter(keys, usr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-API-Key", "boss_wrong_key")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401", w.Code)
	}
}

func TestAPIKeyAuthMissingFallsThroughToJWT(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]int64{}}
	usr := &fakeUserService{profiles: map[int64]*user.Profile{}}
	r := newAPIKeyRouter(keys, usr)

	// 无 API key → 回退 JWT,Authn(nil) 无法验证 → 401
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401(missing bearer)", w.Code)
	}
}

func TestAPIKeyAuthAccountNotFound(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]int64{sha256Hash("boss_orphan_key"): 999}}
	usr := &fakeUserService{profiles: map[int64]*user.Profile{}}
	r := newAPIKeyRouter(keys, usr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-API-Key", "boss_orphan_key")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401 (account not found)", w.Code)
	}
}