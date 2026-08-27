package adminapi

// 契约:/site/posts 官网匿名读(免 token,仅 PUBLISHED 投影)与
// /site-posts 管理端 CRUD(menu:site 权限)。

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakeCMS struct {
	listed  []cms.Post
	created cms.Post
	updated bool
	deleted bool
	pubLang string // ListPublished 收到的 lang(000155 透传断言)
}

func (f *fakeCMS) ListPosts(context.Context) ([]cms.Post, error) { return f.listed, nil }

func (f *fakeCMS) ListPublished(_ context.Context, _ string, lang string, limit int) ([]cms.Post, error) {
	f.pubLang = lang
	return f.listed[:min(limit, len(f.listed))], nil
}

func (f *fakeCMS) GetPublishedBySlug(_ context.Context, slug, lang string) (*cms.Post, error) {
	if p, ok := f.findPublished(slug, lang); ok {
		return p, nil
	}
	if lang != cms.LangDefault {
		if p, ok := f.findPublished(slug, cms.LangDefault); ok {
			return p, nil
		}
	}
	return nil, cms.ErrPostNotFound
}

func (f *fakeCMS) findPublished(slug, lang string) (*cms.Post, bool) {
	for i := range f.listed {
		p := &f.listed[i]
		if p.Slug == slug && p.Lang == lang && p.Status == cms.StatusPublished {
			return p, true
		}
	}
	return nil, false
}

func (f *fakeCMS) CreatePost(_ context.Context, p cms.Post) (int64, error) {
	f.created = p
	return 7, nil
}

func (f *fakeCMS) UpdatePost(_ context.Context, p cms.Post) error { f.updated = true; return nil }

func (f *fakeCMS) DeletePost(_ context.Context, id int64) error { f.deleted = true; return nil }

func newSiteTestRouter(f *fakeUser, cmsSvc cms.Service) (*gin.Engine, *fakeCMS) {
	fc, ok := cmsSvc.(*fakeCMS)
	if !ok {
		fc = &fakeCMS{}
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: f, CMS: fc}, auth.NewManager("t", time.Hour))
	return r, fc
}

// 官网匿名列表:无 token 200,投影不含 version/authorName 管理字段。
func TestSitePublicList_NoAuth_MinimalProjection(t *testing.T) {
	r, _ := newSiteTestRouter(&fakeUser{}, &fakeCMS{listed: []cms.Post{{
		ID: 1, Slug: "hello", Lang: "zh-CN", Title: "标题", Category: "NEWS", Status: cms.StatusPublished,
		Version: 9, AuthorName: "内部作者", PublishedAt: "2026-08-28 10:00",
	}}})
	w := getJSON(t, r, "/api/admin/v1/site/posts", "")
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Code != 0 || len(env.Data.Items) != 1 {
		t.Fatalf("env=%+v", env)
	}
	item := env.Data.Items[0]
	if item["slug"] != "hello" || item["version"] != nil || item["authorName"] != nil {
		t.Fatalf("projection leaked admin fields: %v", item)
	}
	if item["lang"] != "zh-CN" || item["categoryName"] != "动态" {
		t.Fatalf("projection missing lang/categoryName: %v", item)
	}
}

// 官网匿名列表:lang 参数透传域层(000155),非法值归一默认。
func TestSitePublicList_LangPassthrough(t *testing.T) {
	r, fc := newSiteTestRouter(&fakeUser{}, &fakeCMS{})
	getJSON(t, r, "/api/admin/v1/site/posts?lang=en-US", "")
	if fc.pubLang != "en-US" {
		t.Fatalf("pubLang=%q want en-US", fc.pubLang)
	}
	getJSON(t, r, "/api/admin/v1/site/posts?lang=fr-FR", "")
	if fc.pubLang != cms.LangDefault {
		t.Fatalf("pubLang=%q want default %q", fc.pubLang, cms.LangDefault)
	}
}

// 官网匿名详情:lang 变体精确命中;缺变体回退默认语言(000155)。
func TestSitePublicDetail_LangVariantAndFallback(t *testing.T) {
	r, _ := newSiteTestRouter(&fakeUser{}, &fakeCMS{listed: []cms.Post{
		{ID: 1, Slug: "a", Lang: "zh-CN", Title: "中文", Status: cms.StatusPublished},
		{ID: 2, Slug: "a", Lang: "en-US", Title: "English", Status: cms.StatusPublished},
	}})
	var env struct {
		Code int `json:"code"`
		Data struct {
			Title string `json:"title"`
			Lang  string `json:"lang"`
		} `json:"data"`
	}
	w := getJSON(t, r, "/api/admin/v1/site/posts/a?lang=en-US", "")
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Data.Title != "English" || env.Data.Lang != "en-US" {
		t.Fatalf("en variant: env=%+v err=%v", env, err)
	}
	w = getJSON(t, r, "/api/admin/v1/site/posts/a?lang=ms-MY", "")
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Data.Title != "中文" || env.Data.Lang != "zh-CN" {
		t.Fatalf("fallback: env=%+v err=%v", env, err)
	}
}

// 管理端:无 menu:site 权限 403。
func TestSiteAdminRoutes_RequireMenuSitePerm(t *testing.T) {
	mgr := auth.NewManager("t", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	r, _ := newSiteTestRouter(&fakeUser{permOk: false}, &fakeCMS{})
	w := getJSON(t, r, "/api/admin/v1/site-posts", token)
	if w.Code != 403 {
		t.Fatalf("status=%d want 403", w.Code)
	}
}

// 管理端:有权限创建成功,透传域字段。
func TestSiteAdminCreate_OK(t *testing.T) {
	mgr := auth.NewManager("t", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	r, fc := newSiteTestRouter(&fakeUser{permOk: true}, &fakeCMS{})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/site-posts",
		strings.NewReader(`{"slug":"hello","title":"标题","summary":"摘要","content":"正文","status":"PUBLISHED"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fc.created.Slug != "hello" || fc.created.Status != cms.StatusPublished {
		t.Fatalf("created=%+v", fc.created)
	}
}

// 公开详情:草稿一律 404 envelope。
func TestSitePublicDetail_DraftIs404(t *testing.T) {
	r, _ := newSiteTestRouter(&fakeUser{}, &fakeCMS{listed: []cms.Post{{
		ID: 1, Slug: "draft", Status: cms.StatusDraft,
	}}})
	w := getJSON(t, r, "/api/admin/v1/site/posts/draft", "")
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != 40400 {
		t.Fatalf("code=%d want 40400", env.Code)
	}
}

func (f *fakeCMS) ListCategories(context.Context) ([]cms.Category, error) {
	return []cms.Category{{ID: 1, Code: "NEWS", Name: "动态", Enabled: true}}, nil
}
func (f *fakeCMS) CreateCategory(_ context.Context, c cms.Category) (int64, error) { return 5, nil }
func (f *fakeCMS) UpdateCategory(_ context.Context, c cms.Category) error          { return nil }
func (f *fakeCMS) DeleteCategory(_ context.Context, id int64) error                { return nil }
