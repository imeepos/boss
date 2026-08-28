package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// upload 附件上传(multipart,字段 file): bossctl upload [--portal admin|user|worker] FILE
// 三端路径 /attachments/upload;身份由 API key/JWT 决定,上传者类型随主体落 attachments。
func (c *CLI) upload(args []string) error {
	portal, file := "admin", ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--portal" && i+1 < len(args) {
			portal = args[i+1]
			i++
			continue
		}
		file = args[i]
	}
	if file == "" {
		return fmt.Errorf("用法: bossctl upload [--portal admin|user|worker] FILE(上限 32MB)")
	}

	prefix := ""
	for _, p := range portalPrefixes {
		if p.Name == portal {
			prefix = p.Prefix
		}
	}
	if prefix == "" {
		return fmt.Errorf("未知端: %s(可用 admin/user/worker)", portal)
	}

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("打开文件: %w", err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("file", filepath.Base(file))
		if err == nil {
			_, err = io.Copy(part, f)
		}
		if err == nil {
			err = mw.Close()
		}
		pw.CloseWithError(err)
	}()

	url := c.cfg.Server + prefix + "/attachments/upload"
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if name, value := c.authHeaderValue(); name != "" {
		req.Header.Set(name, value)
	}

	resp, err := uploadClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	ar, err := decodeEnvelope(body)
	if err != nil {
		return fmt.Errorf("响应 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if ar.Code != 0 {
		return ar.errBiz("上传")
	}
	printJSON(ar.Data)
	return nil
}
