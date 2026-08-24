package adminapi

// 契约:公开正文图片流 /site/posts/:slug/img/:attId —— 仅"已发布正文确实引用"的
// image/* 附件;未被引用/草稿/非图片一律 404。详情端点把 att/N 重写为该 URL。

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/cms"
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
