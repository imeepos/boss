package pdfgen

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestEscape(t *testing.T) {
	if got := escape(`a(b)c\d`); got != `a\(b\)c\\d` {
		t.Fatalf("escape: %q", got)
	}
	if got := escape("plain"); got != "plain" {
		t.Fatalf("escape plain: %q", got)
	}
}

func TestTextOps(t *testing.T) {
	ops := textOps("In(v)o\\ice", []string{"line 1", "line(2)"})
	s := string(ops)
	if !strings.Contains(s, `(In\(v\)o\\ice) Tj`) {
		t.Fatalf("title missing: %q", s)
	}
	if !strings.Contains(s, `(line\(2\)) Tj`) {
		t.Fatalf("line escape missing: %q", s)
	}
	if n := strings.Count(s, " Tj "); n != 3 {
		t.Fatalf("want 3 Tj ops, got %d", n)
	}
	if ops2 := textOps("T", nil); strings.Count(string(ops2), " Tj ") != 1 {
		t.Fatal("empty lines should emit only title op")
	}
}

func TestBuild(t *testing.T) {
	pdf := Build("凭证", []string{"a(b)", `c\d`})
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4\n")) {
		t.Fatal("missing PDF header")
	}
	if !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
		t.Fatal("missing EOF marker")
	}
	s := string(pdf)
	if !strings.Contains(s, "/Type /Catalog") || !strings.Contains(s, "/BaseFont /Helvetica") {
		t.Fatal("missing catalog/font object")
	}
	if !strings.Contains(s, "xref\n0 6\n") {
		t.Fatal("xref should list 6 entries")
	}
	if !strings.Contains(s, `(a\(b\)) Tj`) || !strings.Contains(s, `(c\\d) Tj`) {
		t.Fatal("content stream missing escaped lines")
	}
	// /Length 必须等于内容流字节数:反解 stream 体核对。
	begin := strings.Index(s, "stream\n") + len("stream\n")
	end := strings.Index(s, "\nendstream")
	var length int
	if _, err := fmt.Sscanf(s[strings.Index(s, "/Length "):], "/Length %d", &length); err != nil {
		t.Fatalf("parse /Length: %v", err)
	}
	if length != end-begin {
		t.Fatalf("/Length=%d, stream body=%d", length, end-begin)
	}
	if pdf2 := Build("T", nil); !bytes.HasSuffix(pdf2, []byte("%%EOF\n")) {
		t.Fatal("empty-lines build broken")
	}
}
