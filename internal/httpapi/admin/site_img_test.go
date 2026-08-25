package adminapi

// 契约:公开正文图片流 /site/posts/:slug/img/:attId —— 仅"已发布正文确实引用"的
// image/* 附件;未被引用/草稿/非图片一律 404。详情端点把 att/N 重写为该 URL。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func newImgRouter(content string, status string, at *attachment.Attachment) *gin.Engine {
	return newContentRouter(content, status, at)
}

func TestSiteImg_OnlyReferencedImageOfPublishedPost(t *testing.T) {
	img := &attachment.Attachment{ID: 9, FileName: "i.png", ContentType: "image/png"}

	cases := []struct {
		name    string
		content string
		status  string
		at      *attachment.Attachment
		want    int
	}{
		{"referenced image", "![](att/9)", cms.StatusPublished, img, 200},
		{"not referenced", "![](att/8)", cms.StatusPublished, img, 404},
		{"draft hides", "![](att/9)", cms.StatusDraft, img, 404},
		{"non-image ref", "![](att/9)", cms.StatusPublished,
			&attachment.Attachment{ID: 9, ContentType: "application/pdf"}, 404},
		{"missing attachment", "![](att/9)", cms.StatusPublished, nil, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newImgRouter(tc.content, tc.status, tc.at)
			w := getJSON(t, r, "/api/admin/v1/site/posts/a/img/9", "")
			if w.Code != tc.want {
				t.Fatalf("status=%d want %d body=%s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

// fakeAtStoreMulti 支持多附件的桩(fakeAtStore 只回填单个)。
type fakeAtStoreMulti struct {
	m map[int64]*attachment.Attachment
}

func (f *fakeAtStoreMulti) Create(context.Context, *attachment.Attachment) (*attachment.Attachment, error) {
	return nil, nil
}
func (f *fakeAtStoreMulti) Get(_ context.Context, id int64) (*attachment.Attachment, error) {
	if a := f.m[id]; a != nil {
		return a, nil
	}
	return nil, attachment.ErrNotFound
}
func (f *fakeAtStoreMulti) ListByUploader(context.Context, string, int64, int) ([]attachment.Attachment, error) {
	return nil, nil
}
func (f *fakeAtStoreMulti) List(context.Context, attachment.ListFilter) ([]attachment.Attachment, int, error) {
	return nil, 0, nil
}
func (f *fakeAtStoreMulti) Delete(context.Context, int64) error { return nil }
func (f *fakeAtStoreMulti) GetByIDs(context.Context, []int64) ([]attachment.Attachment, error) {
	return nil, nil
}

// TestSiteImg_CrossSlugAntiEnumeration 契约:附件被 post A 引用但未被 post B 引用时,
// 通过 post B 的 slug 请求该附件应 404(防枚举)。
func TestSiteImg_CrossSlugAntiEnumeration(t *testing.T) {
	img9 := &attachment.Attachment{ID: 9, FileName: "a.png", ContentType: "image/png"}
	img8 := &attachment.Attachment{ID: 8, FileName: "b.png", ContentType: "image/png"}
	store := &fakeAtStoreMulti{m: map[int64]*attachment.Attachment{9: img9, 8: img8}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := &app.Application{
		User: &fakeUser{},
		CMS: &fakeCMS{listed: []cms.Post{
			{ID: 1, Slug: "post-a", Status: cms.StatusPublished, Content: "![x](att/9)"},
			{ID: 2, Slug: "post-b", Status: cms.StatusPublished, Content: "![y](att/8)"},
		}},
		Attachment: &attachment.Service{St: store, Obj: &fakeObjStorage{content: "IMG"}},
	}
	Register(r, a, auth.NewManager("t", time.Hour))

	// post-a 引用 att/9 → 200
	if w := getJSON(t, r, "/api/admin/v1/site/posts/post-a/img/9", ""); w.Code != 200 {
		t.Fatalf("post-a img/9 status=%d want 200", w.Code)
	}
	// post-b 未引用 att/9 → 404(防枚举)
	if w := getJSON(t, r, "/api/admin/v1/site/posts/post-b/img/9", ""); w.Code != 404 {
		t.Fatalf("post-b img/9 status=%d want 404 (cross-slug anti-enumeration)", w.Code)
	}
	// post-b 引用 att/8 → 200
	if w := getJSON(t, r, "/api/admin/v1/site/posts/post-b/img/8", ""); w.Code != 200 {
		t.Fatalf("post-b img/8 status=%d want 200", w.Code)
	}
}

// TestSiteDetail_RewritesAttRefs 详情契约:content 内 ](att/N) 重写为公开图片 URL。
func TestSiteDetail_RewritesAttRefs(t *testing.T) {
	r := newImgRouter("![x](att/9) [y](https://e.x/z)", cms.StatusPublished, nil)
	w := getJSON(t, r, "/api/admin/v1/site/posts/a", "")
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	want := "/api/admin/v1/site/posts/a/img/9"
	if !strings.Contains(w.Body.String(), want) {
		t.Fatalf("body missing %s: %s", want, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "https://e.x/z") {
		t.Fatal("external link must stay untouched")
	}
}
