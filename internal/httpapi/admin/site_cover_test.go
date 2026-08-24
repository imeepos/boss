package adminapi

// 契约:公开封面流 /site/posts/:slug/cover —— 仅 PUBLISHED 文章的封面、仅 image/*,
// 草稿/无封面/非图片一律 404,不暴露附件通用读通道。

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeAtStore 只回填 Get;Service.Download = St.Get + Obj.Open。
type fakeAtStore struct{ at *attachment.Attachment }

func (f *fakeAtStore) Create(context.Context, *attachment.Attachment) (*attachment.Attachment, error) {
	return nil, nil
}
func (f *fakeAtStore) Get(_ context.Context, id int64) (*attachment.Attachment, error) {
	if f.at == nil || id != f.at.ID {
		return nil, attachment.ErrNotFound
	}
	return f.at, nil
}
func (f *fakeAtStore) ListByUploader(context.Context, string, int64, int) ([]attachment.Attachment, error) {
	return nil, nil
}
func (f *fakeAtStore) List(context.Context, attachment.ListFilter) ([]attachment.Attachment, int, error) {
	return nil, 0, nil
}
func (f *fakeAtStore) Delete(context.Context, int64) error { return nil }
func (f *fakeAtStore) GetByIDs(context.Context, []int64) ([]attachment.Attachment, error) {
	return nil, nil
}

type fakeObjStorage struct{ content string }

func (f *fakeObjStorage) Put(context.Context, attachment.MinIOConfig, io.Reader, int64, string, string) (string, error) {
	return "", nil
}
func (f *fakeObjStorage) Open(context.Context, attachment.MinIOConfig, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(f.content)), nil
}

func newCoverRouter(published bool, cover int64, at *attachment.Attachment) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := &app.Application{
		User: &fakeUser{},
		CMS: &fakeCMS{listed: []cms.Post{{ID: 1, Slug: "a",
			Status: coverStatus(published), CoverAttachment: cover}}},
		Attachment: &attachment.Service{St: &fakeAtStore{at: at}, Obj: &fakeObjStorage{content: "IMG"}},
	}
	Register(r, a, auth.NewManager("t", time.Hour))
	return r
}

func coverStatus(pub bool) string {
	if pub {
		return cms.StatusPublished
	}
	return cms.StatusDraft
}

func TestSiteCover_OnlyPublishedWithImageCover(t *testing.T) {
	img := &attachment.Attachment{ID: 9, FileName: "c.png", ContentType: "image/png"}
	pdf := &attachment.Attachment{ID: 9, FileName: "a.pdf", ContentType: "application/pdf"}

	cases := []struct {
		name  string
		pub   bool
		cover int64
		at    *attachment.Attachment
		want  int
	}{
		{"published with image cover", true, 9, img, 200},
		{"draft hides cover", false, 9, img, 404},
		{"no cover 404", true, 0, img, 404},
		{"non-image cover 404", true, 9, pdf, 404},
		{"missing attachment 404", true, 9, nil, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newCoverRouter(tc.pub, tc.cover, tc.at)
			w := getJSON(t, r, "/api/admin/v1/site/posts/a/cover", "")
			if w.Code != tc.want {
				t.Fatalf("status=%d want %d body=%s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
