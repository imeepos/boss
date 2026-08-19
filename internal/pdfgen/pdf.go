// Package pdfgen 极简中文无关 PDF 生成器:单页 A4 + Helvetica 文本行,
// 仅服务门户凭证/发票下载,不引入外部依赖(vendor 纪律见 docs/notes/adopted)。
package pdfgen

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	pageW, pageH         = 595.28, 841.89 // A4 pt
	fontSize             = 11
	titleSize            = 16
	lineGap              = 20
	marginX      float64 = 60
	marginTop    float64 = 760
)

// escape PDF 字符串转义((),\)。
func escape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return r.Replace(s)
}

// textOps 生成内容流:首行标题,其余按固定行距排列。
func textOps(title string, lines []string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "BT /F1 %d Tf %g %g Td (%s) Tj ET\n",
		titleSize, marginX, marginTop, escape(title))
	y := marginTop - lineGap*1.4
	for _, ln := range lines {
		fmt.Fprintf(&b, "BT /F1 %d Tf %g %g Td (%s) Tj ET\n", fontSize, marginX, y, escape(ln))
		y -= lineGap
	}
	return b.Bytes()
}

// Build 组装单页 PDF 字节(对象表 + xref,PDF 1.4)。
func Build(title string, lines []string) []byte {
	content := textOps(title, lines)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %g %g] "+
			"/Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>", pageW, pageH),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xref)
	return out.Bytes()
}
