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
	// 初始化核心
	core.Init()
	
	addr := ":9876"
	if v := os.Getenv("MC_LAUNCHER_ADDR"); v != "" {
		addr = v
	}
	
	// 准备静态文件
	staticDir := filepath.Join(core.GetBaseDir(), "static")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		// 尝试从项目目录复制
		projectDir, _ := os.Getwd()
		frontendDir := filepath.Join(projectDir, "frontend")
		if info, err := os.Stat(frontendDir); err == nil && info.IsDir() {
			os.MkdirAll(staticDir, 0755)
			// 简单复制（实际应该用更健壮的方式）
			fmt.Println("警告: 未找到静态文件目录，请运行安装脚本")
		}
	}
	
	mux := http.NewServeMux()
	
	// API 路由
	mux.HandleFunc("/api/ping", handlePing)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/versions", handleVersions)
	mux.HandleFunc("/api/versions/install", handleInstallVersion)
	mux.HandleFunc("/api/versions/remove", handleRemoveVersion)
	mux.HandleFunc("/api/launch", handleLaunch)
	mux.HandleFunc("/api/accounts", handleAccounts)
	mux.HandleFunc("/api/accounts/offline", handleOfflineAuth)
	mux.HandleFunc("/api/accounts/microsoft", handleMicrosoftAuth)
	mux.HandleFunc("/api/java", handleJava)
	mux.HandleFunc("/api/java/install", handleInstallJava)
	mux.HandleFunc("/api/java/set", handleSetJava)
	mux.HandleFunc("/api/modrinth/search", handleModrinthSearch)
	mux.HandleFunc("/api/modrinth/info/", handleModrinthInfo)
	mux.HandleFunc("/api/config", handleConfig)
	
	// 静态文件服务
	if _, err := os.Stat(staticDir); err == nil {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	} else {
		// 默认返回 index.html 或直接提供 API
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "ok",
				"message": "MC Launcher WebUI Server",
				"version": "2.0.0-dev",
			})
		})
	}
	
	fmt.Printf("MC Launcher WebUI Server\n")
	fmt.Printf("Address: %s\n", addr)
	fmt.Printf("Base Dir: %s\n", core.GetBaseDir())
	fmt.Println("================================")
	
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ============ 响应助手 ============

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeResponse(w http.ResponseWriter, resp *core.Response) {
	status := http.StatusOK
	if !resp.Ok {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, resp)
}

func okResp(data any) *core.Response {
	return &core.Response{Ok: true, Data: data}
}

func errResp(msg string) *core.Response {
	return &core.Response{Ok: false, Error: msg}
}

// ============ API Handlers ============

func handlePing(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, okResp(map[string]string{"status": "ok", "version": "2.0.0"}))
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, okResp(map[string]any{
		"version":  "2.0.0",
		"baseDir":  core.GetBaseDir(),
		"accounts": len(core.ListAccounts()),
	}))
}

func handleVersions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		writeResponse(w, okResp(core.ListVersions()))
	default:
		writeResponse(w, errResp("method not allowed"))
	}
}

func handleInstallVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	var req struct {
		Version string `json:"version"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	
	if req.Version == "" {
		writeResponse(w, errResp("version required"))
		return
	}
	
	resp := core.InstallVersion(req.Version, "", nil)
	writeResponse(w, resp)
}

func handleRemoveVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	var req struct {
		Version string `json:"version"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	
	err := core.RemoveVersion(req.Version)
	if err != nil {
		writeResponse(w, errResp(err.Error()))
		return
	}
	writeResponse(w, okResp(nil))
}

func handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	var req struct {
		Version  string `json:"version"`
		Username string `json:"username"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	
	if req.Version == "" {
		writeResponse(w, errResp("version required"))
		return
	}
	if req.Username == "" {
		req.Username = "Player"
	}
	
	resp := core.LaunchVersion(req.Version, req.Username)
	writeResponse(w, resp)
}

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		writeResponse(w, okResp(core.ListAccounts()))
	case "DELETE":
		core.AccountRemove("")
		writeResponse(w, okResp(nil))
	default:
		writeResponse(w, errResp("method not allowed"))
	}
}

func handleOfflineAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	
	if req.Name == "" {
		req.Name = "Player"
	}
	
	acc := core.LoginOffline(req.Name)
	core.AccountUpsert(*acc)
	writeResponse(w, okResp(acc))
}

func handleMicrosoftAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	acc, err := core.LoginMicrosoft(true, "")
	if err != nil {
		writeResponse(w, errResp(err.Error()))
		return
	}
	core.AccountUpsert(*acc)
	writeResponse(w, okResp(acc))
}

func handleJava(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		writeResponse(w, okResp(core.ScanJava()))
	default:
		writeResponse(w, errResp("method not allowed"))
	}
}

func handleInstallJava(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	var req struct {
		Major int `json:"major"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	
	if req.Major == 0 {
		req.Major = 17
	}
	
	path, err := core.InstallJava(req.Major)
	if err != nil {
		writeResponse(w, errResp(err.Error()))
		return
	}
	writeResponse(w, okResp(map[string]string{"path": path}))
}

func handleSetJava(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		writeResponse(w, errResp("method not allowed"))
		return
	}
	
	var req struct {
		Path string `json:"path"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	
	if req.Path == "" {
		writeResponse(w, errResp("path required"))
		return
	}
	
	core.SetJavaPath(req.Path)
	writeResponse(w, okResp(nil))
}

func handleModrinthSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	projectType := r.URL.Query().Get("type")
	offsetStr := r.URL.Query().Get("offset")
	
	offset := 0
	if n, err := strconv.Atoi(offsetStr); err == nil {
		offset = n
	}
	
	resp := core.ModrinthSearch(query, projectType, offset)
	writeResponse(w, resp)
}

func handleModrinthInfo(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/api/modrinth/info/")
	if slug == "" {
		writeResponse(w, errResp("slug required"))
		return
	}
	
	resp := core.ModrinthProjectInfo(slug)
	writeResponse(w, resp)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		writeResponse(w, okResp(core.GetSettings()))
	case "PUT":
		var updated core.Settings
		json.NewDecoder(r.Body).Decode(&updated)
		c := core.GetSettings()
		if updated.MaxMemory > 0 {
			c.MaxMemory = updated.MaxMemory
		}
		if updated.MinMemory > 0 {
			c.MinMemory = updated.MinMemory
		}
		if updated.Threads > 0 {
			c.Threads = updated.Threads
		}
		if updated.Width > 0 {
			c.Width = updated.Width
		}
		if updated.Height > 0 {
			c.Height = updated.Height
		}
		core.SaveSettings(c)
		writeResponse(w, okResp(c))
	default:
		writeResponse(w, errResp("method not allowed"))
	}
}
