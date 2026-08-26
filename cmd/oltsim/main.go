// OLT 设备仿真接入程序:模拟厂家 OLT CLI(管理面)+ 光猫上线(RADIUS 客户端)。
// 用途:测试 BOSS→设备 连通性与业务完整性(见同目录 README.md)。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// timeNow 秒级时间戳(台账展示)。
func timeNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func main() {
	var (
		telnetAddr = flag.String("telnet", ":2323", "OLT Telnet CLI 监听地址")
		httpAddr   = flag.String("http", ":8081", "模拟器管理面 HTTP 地址")
		user       = flag.String("user", "admin", "OLT CLI 登录用户名")
		pass       = flag.String("pass", "admin", "OLT CLI 登录密码")
		denyLogin  = flag.Bool("deny-login", false, "故障注入:登录一律拒绝")
		forceErr   = flag.Bool("force-err", false, "故障注入:下发命令一律回 ERR")
		radiusAddr = flag.String("radius", "127.0.0.1:1812", "boss-aaa RADIUS 鉴权地址")
		radiusSec  = flag.String("radius-secret", "boss-aaa-secret", "RADIUS 共享密钥")
	)
	flag.Parse()

	sim := &Sim{
		TelnetAddr: *telnetAddr, User: *user, Pass: *pass,
		DenyLogin: *denyLogin, ForceErr: *forceErr,
		RadiusAddr: *radiusAddr, RadiusSecret: *radiusSec,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ln, err := listenTCP(*telnetAddr)
	if err != nil {
		log.Fatalf("oltsim: listen %s: %v", *telnetAddr, err)
	}
	go sim.serveTelnet(ctx, ln)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /records", sim.handleRecords)
	mux.HandleFunc("POST /ont/online", sim.handleOntOnline)
	hs := &http.Server{Addr: *httpAddr, Handler: mux}
	go func() {
		log.Printf("oltsim: admin http on %s", *httpAddr)
		if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("oltsim: http: %v", err)
		}
	}()

	log.Printf("oltsim: telnet on %s (user=%s deny=%v forceErr=%v)", *telnetAddr, *user, *denyLogin, *forceErr)
	<-ctx.Done()
	_ = hs.Shutdown(context.Background())
	_ = ln.Close()
	log.Println("oltsim: stopped")
}

// Sim 仿真器状态(线程安全:台账经 mutex)。
type Sim struct {
	TelnetAddr   string
	User         string
	Pass         string
	DenyLogin    bool
	ForceErr     bool
	RadiusAddr   string
	RadiusSecret string

	mu      sync.Mutex
	records []Record
}

// Record 一笔下发/上线记录(管理面可查,业务完整性证据)。
type Record struct {
	Type     string `json:"type"` // provision/ont-online
	Template string `json:"template,omitempty"`
	TaskNo   string `json:"taskNo,omitempty"`
	Event    string `json:"event,omitempty"`
	Loid     string `json:"loid,omitempty"`
	Result   string `json:"result"`           // OK/ERR/ACCEPT/REJECT
	Detail   string `json:"detail,omitempty"` // 带宽等补充
	At       string `json:"at"`
}

func (s *Sim) add(r Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.At = timeNow()
	s.records = append(s.records, r)
}

func (s *Sim) all() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Record(nil), s.records...)
	return out
}

// handleRecords 管理面:下发/上线台账。
func (s *Sim) handleRecords(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ginH{"items": s.all()})
}

// handleOntOnline 光猫上线:向 boss-aaa 发 RADIUS Access-Request。
func (s *Sim) handleOntOnline(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Loid string `json:"loid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Loid == "" {
		writeJSON(w, http.StatusBadRequest, ginH{"error": "loid required"})
		return
	}
	code, bw, err := s.radiusAuth(r.Context(), req.Loid)
	rec := Record{Type: "ont-online", Loid: req.Loid, Detail: bw}
	switch {
	case err != nil:
		rec.Result = "ERROR"
		rec.Detail = err.Error()
	case code == codeAccessAccept:
		rec.Result = "ACCEPT"
	default:
		rec.Result = "REJECT"
	}
	s.add(rec)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, ginH{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ginH{"loid": req.Loid, "result": rec.Result, "bandwidth": bw})
}

// lint 兜底类型(JSON 响应辅助)。
type ginH map[string]any

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
