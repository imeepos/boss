package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	maxBodyBytes    = 1 << 20
	healthTimeout   = 30 * time.Second
	healthPollEvery = 2 * time.Second
	maxClockSkew    = 5 * time.Minute
)

type config struct {
	Listen      string
	SecretFile  string
	ComposeDir  string
	ComposeFile string
	Project     string
	MinioCtr    string
}

var (
	cfg      *config
	rotateMu sync.Mutex
)

// auth 校验 HMAC 头与时间戳窗口。
func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := r.Header.Get("X-Hostctl-Timestamp")
		sig := r.Header.Get("X-Hostctl-Signature")
		if ts == "" || sig == "" {
			writeErr(w, http.StatusUnauthorized, "missing auth headers")
			return
		}
		var t int64
		if _, err := fmt.Sscanf(ts, "%d", &t); err != nil {
			writeErr(w, http.StatusUnauthorized, "bad timestamp")
			return
		}
		now := time.Now().Unix()
		skew := now - t
		if skew > int64(maxClockSkew/time.Second) || skew < -int64(maxClockSkew/time.Second) {
			writeErr(w, http.StatusUnauthorized, "timestamp out of window")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "read body")
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(body)))

		secret, err := os.ReadFile(cfg.SecretFile)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "secret read fail")
			return
		}
		mac := hmac.New(sha256.New, []byte(strings.TrimSpace(string(secret))))
		fmt.Fprintf(mac, "%s\n%s\n%s\n%s", ts, r.Method, r.URL.Path, string(body))
		got := mac.Sum(nil)
		want, err := hex.DecodeString(sig)
		if err != nil || !hmac.Equal(got, want) {
			writeErr(w, http.StatusUnauthorized, "bad signature")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{"ok": true, "ts": time.Now().Unix()})
}

type rotateReq struct {
	Secret string `json:"secret"`
}

type rotateResp struct {
	OK        bool   `json:"ok"`
	RotatedAt string `json:"rotatedAt"`
	Duration  string `json:"duration"`
}

func handleRotateSecret(w http.ResponseWriter, r *http.Request) {
	rotateMu.Lock()
	defer rotateMu.Unlock()

	var req rotateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	if n := len(req.Secret); n < 8 || n > 128 {
		writeErr(w, http.StatusBadRequest, "secret length must be 8..128")
		return
	}

	start := time.Now()
	if err := writeSecretFile(req.Secret); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := restartMinio(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !waitMinioHealthy(r.Context()) {
		writeErr(w, http.StatusInternalServerError, "minio health not ready after 30s")
		return
	}
	writeOK(w, rotateResp{
		OK:        true,
		RotatedAt: time.Now().UTC().Format(time.RFC3339),
		Duration:  time.Since(start).String(),
	})
}

func writeSecretFile(secret string) error {
	// 进程以非 root 运行:先写 /tmp/hostctl.secret(0600),再 sudo install 到目标路径。
	// sudoers 白名单只允许该精确命令,无通配。
	tmp := "/tmp/hostctl.secret"
	if err := os.WriteFile(tmp, []byte(secret), 0600); err != nil {
		return errors.New("write tmp: " + err.Error())
	}
	cmd := exec.Command("sudo", "-n",
		"/usr/bin/install", "-m", "0600", "-o", "root", "-g", "root",
		tmp, "/etc/minio/secrets/root_password")
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("sudo install: " + err.Error() + " | " + string(cmdOut))
	}
	_ = os.Remove(tmp)
	return nil
}

func restartMinio(ctx context.Context) error {
	args := []string{"compose", "-p", cfg.Project}
	for _, f := range strings.Split(cfg.ComposeFile, ",") {
		args = append(args, "-f", strings.TrimSpace(f))
	}
	args = append(args, "up", "-d", "--force-recreate", "--no-deps", "minio")
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = cfg.ComposeDir
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose restart: %v | %s", err, string(cmdOut))
	}
	return nil
}

func waitMinioHealthy(ctx context.Context) bool {
	deadline := time.Now().Add(healthTimeout)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return false
		}
		c := exec.CommandContext(ctx, "curl", "-fsS", "-o", "/dev/null",
			"-w", "%{http_code}", "--max-time", "2",
			"http://127.0.0.1:29000/minio/health/live")
		out, err := c.CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "200" {
			return true
		}
		time.Sleep(healthPollEvery)
	}
	return false
}

func writeOK(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
}
