package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// release 客户端版本发布域 CLI:上传 APK 建版 + 状态/灰度管理 + 列表。
// bossctl release upload --app user --version 1.2.0 --code 12 [--min 10]
//
//	[--notes S] [--status DRAFT|GRAY|PUBLISHED] [--rollout N] [--whitelist 1,2] FILE.apk
//
// bossctl release patch <id> --status GRAY --rollout 20 [--whitelist 1,2] [--min 10] [--notes S] [--force]
// bossctl release list [--app user|worker]
func (c *CLI) release(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: bossctl release <upload|patch|list> ...")
	}
	switch args[0] {
	case "upload":
		return c.releaseUpload(args[1:])
	case "patch":
		return c.releasePatch(args[1:])
	case "list":
		return c.releaseList(args[1:])
	default:
		return fmt.Errorf("未知子命令: %s(upload|patch|list)", args[0])
	}
}

type releaseOpts struct {
	app       string
	version   string
	code      int
	minCode   int
	notes     string
	status    string
	rollout   int
	whitelist string
	force     bool
	file      string
	id        int64
}

// parseReleaseFlags 解析公共 flag;file/id 为位置参数。
func parseReleaseFlags(args []string) (*releaseOpts, error) {
	o := &releaseOpts{status: "", minCode: 0}
	for i := 0; i < len(args); i++ {
		next := func() string {
			i++
			if i >= len(args) {
				return ""
			}
			return args[i]
		}
		switch args[i] {
		case "--app":
			o.app = next()
		case "--version":
			o.version = next()
		case "--code":
			n, _ := strconv.Atoi(next())
			o.code = n
		case "--min":
			n, _ := strconv.Atoi(next())
			o.minCode = n
		case "--notes":
			o.notes = next()
		case "--status":
			o.status = next()
		case "--rollout":
			n, _ := strconv.Atoi(next())
			o.rollout = n
		case "--whitelist":
			o.whitelist = next()
		case "--force":
			o.force = true
		default:
			if o.file == "" && !strings.HasPrefix(args[i], "-") {
				o.file = args[i]
			}
		}
	}
	if o.id, _ = strconv.ParseInt(o.file, 10, 64); o.id > 0 {
		o.file = ""
	}
	return o, nil
}

// releaseUpload multipart 上传 APK 创建发版(对接 POST /client-releases)。
func (c *CLI) releaseUpload(args []string) error {
	o, _ := parseReleaseFlags(args)
	if o.file == "" || o.app == "" || o.version == "" || o.code <= 0 {
		return fmt.Errorf("用法: release upload --app user|worker --version 1.2.0 --code 12 [--min N] [--notes S] [--status S] [--rollout N] [--whitelist 1,2] FILE.apk")
	}
	if o.status == "" {
		o.status = "DRAFT"
	}
	f, err := os.Open(o.file)
	if err != nil {
		return fmt.Errorf("打开 APK: %w", err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		var werr error
		if part, e := mw.CreateFormFile("file", filepath.Base(o.file)); e == nil {
			_, werr = io.Copy(part, f)
		} else {
			werr = e
		}
		for k, v := range map[string]string{
			"app": o.app, "version": o.version, "versionCode": strconv.Itoa(o.code),
			"minSupportedCode": strconv.Itoa(o.minCode), "notes": o.notes,
			"status": o.status, "rolloutPercent": strconv.Itoa(o.rollout),
			"whitelistIds": o.whitelist, "force": strconv.FormatBool(o.force),
		} {
			if e := mw.WriteField(k, v); e != nil && werr == nil {
				werr = e
			}
		}
		if werr == nil {
			werr = mw.Close()
		}
		pw.CloseWithError(werr)
	}()

	url := c.cfg.Server + "/api/admin/v1/client-releases"
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if name, value := c.authHeaderValue(); name != "" {
		req.Header.Set(name, value)
	}
	return c.doReleaseRequest(req)
}

// releasePatch 状态/灰度编辑(对接 PATCH /client-releases/:id)。
func (c *CLI) releasePatch(args []string) error {
	o, _ := parseReleaseFlags(args)
	if o.id <= 0 {
		return fmt.Errorf("用法: release patch <id> --status GRAY --rollout 20 [--whitelist 1,2] [--min N] [--notes S] [--force]")
	}
	body := map[string]any{}
	if o.version != "" {
		body["version"] = o.version
	}
	if o.status != "" {
		body["status"] = o.status
	}
	if o.rollout > 0 {
		body["rolloutPercent"] = o.rollout
	}
	if o.minCode > 0 {
		body["minSupportedCode"] = o.minCode
	}
	if o.notes != "" {
		body["notes"] = o.notes
	}
	if o.whitelist != "" {
		ids := []int64{}
		for _, p := range strings.Split(o.whitelist, ",") {
			if v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
				ids = append(ids, v)
			}
		}
		body["whitelistIds"] = ids
	}
	if o.force {
		body["force"] = true
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("PATCH",
		fmt.Sprintf("%s/api/admin/v1/client-releases/%d", c.cfg.Server, o.id),
		strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if name, value := c.authHeaderValue(); name != "" {
		req.Header.Set(name, value)
	}
	return c.doReleaseRequest(req)
}

// releaseList 列表(对接 GET /client-releases)。
func (c *CLI) releaseList(args []string) error {
	q := ""
	if len(args) == 2 && args[0] == "--app" {
		q = "?app=" + args[1]
	}
	req, err := http.NewRequest("GET", c.cfg.Server+"/api/admin/v1/client-releases"+q, nil)
	if err != nil {
		return err
	}
	if name, value := c.authHeaderValue(); name != "" {
		req.Header.Set(name, value)
	}
	return c.doReleaseRequest(req)
}

// doReleaseRequest 发请求并原样打印信封 data。
func (c *CLI) doReleaseRequest(req *http.Request) error {
	resp, err := http.DefaultClient.Do(req)
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
		fmt.Printf("失败: code=%d msg=%s\n", ar.Code, ar.Msg)
		os.Exit(1)
	}
	printJSON(ar.Data)
	return nil
}
