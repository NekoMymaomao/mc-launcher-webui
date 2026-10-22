package core

// modrinth.go — Modrinth API（按官方文档实现）
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const modrinthAPI = "https://api.modrinth.com/v2"

// ModrinthHit 搜索结果条目（字段名与官方一致）
type ModrinthHit struct {
	ProjectID     string   `json:"project_id"`
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	ProjectType   string   `json:"project_type"`
	IconURL       string   `json:"icon_url"`
	Downloads     int64    `json:"downloads"`
	Follows       int64    `json:"follows"`
	Categories    []string `json:"categories"`
	Versions      []string `json:"versions"`
	DateModified  string   `json:"date_modified"`
	LatestVersion string   `json:"latest_version"`
}

// ModrinthSearchResult 搜索响应
type ModrinthSearchResult struct {
	Hits       []ModrinthHit `json:"hits"`
	Offset     int           `json:"offset"`
	Limit      int           `json:"limit"`
	TotalHits  int           `json:"total_hits"`
}

// ModrinthVersion 版本信息
type ModrinthVersion struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	VersionNumber string   `json:"version_number"`
	GameVersions  []string `json:"game_versions"`
	Loaders       []string `json:"loaders"`
	Files         []struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Primary  bool   `json:"primary"`
		Size     int64  `json:"size"`
	} `json:"files"`
}

func modrinthGet(path string, out any) error {
	req, err := http.NewRequest("GET", modrinthAPI+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mc-launcher/2.0 (webui)")
	req.Header.Set("Accept", "application/json")
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("Modrinth HTTP %d: %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ModrinthSearch 搜索项目。projectType 为空表示全部，可选 mod/modpack/shader/resourcepack/datapack
func ModrinthSearch(query, projectType string, offset int) *Response {
	if offset < 0 {
		offset = 0
	}
	vals := url.Values{}
	vals.Set("query", query)
	vals.Set("limit", "20")
	vals.Set("offset", fmt.Sprint(offset))
	if projectType != "" {
		// 官方用 facets 过滤类型，而不是 type 参数
		facets, _ := json.Marshal([][]string{{"project_type:" + projectType}})
		vals.Set("facets", string(facets))
	}

	var res ModrinthSearchResult
	if err := modrinthGet("/search?"+vals.Encode(), &res); err != nil {
		return &Response{Ok: false, Error: err.Error()}
	}
	if res.Hits == nil {
		res.Hits = []ModrinthHit{}
	}
	return &Response{Ok: true, Data: res}
}

// ModrinthProject 获取项目详情
func ModrinthProject(slug string) *Response {
	if slug == "" {
		return &Response{Ok: false, Error: "slug 不能为空"}
	}
	var p map[string]any
	if err := modrinthGet("/project/"+url.PathEscape(slug), &p); err != nil {
		return &Response{Ok: false, Error: err.Error()}
	}
	return &Response{Ok: true, Data: p}
}

// ModrinthVersions 获取项目版本列表（按游戏版本/加载器过滤）
func ModrinthVersions(slug, gameVersion, loader string) *Response {
	if slug == "" {
		return &Response{Ok: false, Error: "slug 不能为空"}
	}
	vals := url.Values{}
	if gameVersion != "" {
		gv, _ := json.Marshal([]string{gameVersion})
		vals.Set("game_versions", string(gv))
	}
	if loader != "" {
		ld, _ := json.Marshal([]string{loader})
		vals.Set("loaders", string(ld))
	}
	q := ""
	if len(vals) > 0 {
		q = "?" + vals.Encode()
	}
	var vs []ModrinthVersion
	if err := modrinthGet("/project/"+url.PathEscape(slug)+"/version"+q, &vs); err != nil {
		return &Response{Ok: false, Error: err.Error()}
	}
	return &Response{Ok: true, Data: vs}
}

// ModrinthInstall 下载并安装到对应目录
// instance 为空时使用 baseDir；contentType: mods|shaderpacks|resourcepacks|datapacks
func ModrinthInstall(slug, gameVersion, loader, contentType, instanceDir string) *Response {
	vr := ModrinthVersions(slug, gameVersion, loader)
	if !vr.Ok {
		return vr
	}
	vs, _ := vr.Data.([]ModrinthVersion)
	if len(vs) == 0 {
		return &Response{Ok: false, Error: "没有匹配的版本"}
	}

	// 选主文件
	var dlURL, filename string
	for _, f := range vs[0].Files {
		if f.Primary {
			dlURL, filename = f.URL, f.Filename
			break
		}
	}
	if dlURL == "" && len(vs[0].Files) > 0 {
		dlURL, filename = vs[0].Files[0].URL, vs[0].Files[0].Filename
	}
	if dlURL == "" || filename == "" {
		return &Response{Ok: false, Error: "该版本没有可下载文件"}
	}

	if contentType == "" {
		contentType = "mods"
	}
	targetDir := filepath.Join(baseDir, contentType)
	if instanceDir != "" {
		targetDir = filepath.Join(instanceDir, contentType)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return &Response{Ok: false, Error: err.Error()}
	}

	dest := filepath.Join(targetDir, filename)
	if err := downloadFile(dlURL, dest, ""); err != nil {
		return &Response{Ok: false, Error: err.Error()}
	}
	return &Response{Ok: true, Data: map[string]string{
		"file":    dest,
		"version": vs[0].VersionNumber,
		"name":    vs[0].Name,
	}}
}
