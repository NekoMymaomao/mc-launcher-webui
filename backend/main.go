package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NekoMymaomao/mc-launcher/core"
)

func main() {
	if err := core.Init(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	addr := os.Getenv("MC_LAUNCHER_ADDR")
	if addr == "" {
		addr = ":9876"
	}

	staticDir := findStaticDir()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, core.Response{Ok: true, Data: map[string]any{
			"version": "2.0.0",
			"baseDir": core.GetBaseDir(),
		}})
	})

	mux.HandleFunc("/api/versions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		writeJSON(w, 200, core.Response{Ok: true, Data: core.ListVersions()})
	})

	mux.HandleFunc("/api/versions/install", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		var body struct {
			Version string `json:"version"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Version == "" {
			writeJSON(w, 400, core.Response{Ok: false, Error: "version 不能为空"})
			return
		}
		resp, err := core.InstallVersion(body.Version, "", nil)
		if err != nil {
			writeJSON(w, 500, core.Response{Ok: false, Error: err.Error()})
			return
		}
		writeJSON(w, 200, resp)
	})

	mux.HandleFunc("/api/versions/remove", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		var body struct {
			Version string `json:"version"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if err := core.RemoveVersion(body.Version); err != nil {
			writeJSON(w, 500, core.Response{Ok: false, Error: err.Error()})
			return
		}
		writeJSON(w, 200, core.Response{Ok: true})
	})

	mux.HandleFunc("/api/versions/verify", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("version")
		if v == "" {
			writeJSON(w, 400, core.Response{Ok: false, Error: "version 不能为空"})
			return
		}
		ok, msg := core.VerifyVersion(v)
		writeJSON(w, 200, core.Response{Ok: ok, Error: msg})
	})

	mux.HandleFunc("/api/launch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		var body struct {
			Version  string `json:"version"`
			Username string `json:"username"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Version == "" {
			writeJSON(w, 400, core.Response{Ok: false, Error: "version 不能为空"})
			return
		}
		if body.Username == "" {
			body.Username = "Player"
		}
		resp, err := core.LaunchVersion(body.Version, body.Username)
		if err != nil {
			writeJSON(w, 500, core.Response{Ok: false, Error: err.Error()})
			return
		}
		writeJSON(w, 200, resp)
	})

	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, core.Response{Ok: true, Data: map[string]any{
				"accounts": core.ListAccounts(),
				"current":  core.CurrentAccount(),
			}})
		case http.MethodDelete:
			uuid := r.URL.Query().Get("uuid")
			if uuid == "" {
				writeJSON(w, 400, core.Response{Ok: false, Error: "uuid 不能为空"})
				return
			}
			ok := core.RemoveAccount(uuid)
			writeJSON(w, 200, core.Response{Ok: ok})
		case http.MethodPut:
			var body struct {
				UUID string `json:"uuid"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			ok := core.SelectAccount(body.UUID)
			writeJSON(w, 200, core.Response{Ok: ok})
		default:
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
		}
	})

	mux.HandleFunc("/api/accounts/offline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name == "" {
			body.Name = "Player"
		}
		acc := core.LoginOffline(body.Name)
		core.UpsertAccount(*acc)
		writeJSON(w, 200, core.Response{Ok: true, Data: acc})
	})

	mux.HandleFunc("/api/accounts/microsoft", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		acc, err := core.LoginMicrosoft(false, "")
		if err != nil {
			writeJSON(w, 400, core.Response{Ok: false, Error: err.Error()})
			return
		}
		core.UpsertAccount(*acc)
		writeJSON(w, 200, core.Response{Ok: true, Data: acc})
	})

	mux.HandleFunc("/api/java", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		writeJSON(w, 200, core.Response{Ok: true, Data: core.ScanJava()})
	})

	mux.HandleFunc("/api/java/install", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		var body struct {
			Major int `json:"major"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Major == 0 {
			body.Major = 17
		}
		p, err := core.InstallJava(body.Major)
		if err != nil {
			writeJSON(w, 500, core.Response{Ok: false, Error: err.Error()})
			return
		}
		writeJSON(w, 200, core.Response{Ok: true, Data: map[string]string{"path": p}})
	})

	mux.HandleFunc("/api/java/set", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
			return
		}
		var body struct {
			Path string `json:"path"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Path == "" {
			writeJSON(w, 400, core.Response{Ok: false, Error: "path 不能为空"})
			return
		}
		if err := core.SetJavaPath(body.Path); err != nil {
			writeJSON(w, 500, core.Response{Ok: false, Error: err.Error()})
			return
		}
		writeJSON(w, 200, core.Response{Ok: true})
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, core.Response{Ok: true, Data: core.GetSettings()})
		case http.MethodPut:
			var s core.Settings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				writeJSON(w, 400, core.Response{Ok: false, Error: err.Error()})
				return
			}
			if err := core.SaveSettings(&s); err != nil {
				writeJSON(w, 500, core.Response{Ok: false, Error: err.Error()})
				return
			}
			writeJSON(w, 200, core.Response{Ok: true, Data: core.GetSettings()})
		default:
			writeJSON(w, 405, core.Response{Ok: false, Error: "method not allowed"})
		}
	})

	mux.HandleFunc("/api/modrinth/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		ptype := r.URL.Query().Get("type")
		off, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		writeJSON(w, 200, core.ModrinthSearch(q, ptype, off))
	})

	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, core.Response{Ok: true, Data: map[string]string{
				"message": "MC Launcher WebUI API",
				"hint":    "未找到 frontend 目录，仅提供 API",
			}})
		})
	}

	fmt.Printf("MC Launcher 服务已启动: http://localhost%s\n", addr)
	fmt.Printf("数据目录: %s\n", core.GetBaseDir())
	if staticDir != "" {
		fmt.Printf("前端目录: %s\n", staticDir)
	}
	log.Fatal(http.ListenAndServe(addr, logRequests(mux)))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// findStaticDir 按优先级查找前端静态目录
func findStaticDir() string {
	candidates := []string{}

	if v := os.Getenv("MC_LAUNCHER_STATIC"); v != "" {
		candidates = append(candidates, v)
	}

	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(d, "static"), filepath.Join(d, "..", "frontend"))
	}

	// 工作目录附近的 frontend
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "frontend"),
			filepath.Join(wd, "..", "frontend"),
			filepath.Join(wd, "static"),
		)
	}

	candidates = append(candidates, filepath.Join(core.GetBaseDir(), "static"))

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
				return c
			}
		}
	}
	return ""
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
