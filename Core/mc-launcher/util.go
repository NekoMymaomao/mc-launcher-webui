package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func launcherDir() string {
	if v := os.Getenv("MC_LAUNCHER_DIR"); v != "" {
		return v
	}
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "bin" {
		dir = filepath.Dir(dir)
	}
	if info, _ := os.Stat(filepath.Join(dir, "launcher")); info != nil {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mc-launcher")
}

func ensureDir(path string) {
	os.MkdirAll(path, 0755)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func listDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func writeAll(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func logInfo(format string, args ...any) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

func initLogger() {}

// mirrorUrl 返回镜像 URL
func mirrorUrl(url, source string) string {
	if source == "bmc" {
		url = strings.Replace(url, "https://libraries.minecraft.net/", "https://mirror.bspapp.com/minecraft/", 1)
		url = strings.Replace(url, "https://resources.download.minecraft.net/", "https://mirror.bspapp.com/minecraft/", 1)
		url = strings.Replace(url, "https://launchermeta.mojang.com/", "https://launcher.mojang.com/", 1)
	}
	return url
}

// downloadFile 下载文件并验证 SHA1
func downloadFile(url, dest, expectedSHA1 string, headers map[string]string) error {
	// 检查是否已存在且校验和匹配
	if expectedSHA1 != "" && fileExists(dest) {
		if sha, err := sha1OfFile(dest); err == nil && sha == expectedSHA1 {
			return nil
		}
	}

	// 确保目标目录存在
	ensureDir(filepath.Dir(dest))

	// 下载文件
	resp, err := http.Get(mirrorUrl(url, cfg.DownloadSource))
	if err != nil {
		return fmt.Errorf("下载失败 %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("创建文件失败 %s: %v", dest, err)
	}
	defer out.Close()

	h := sha1.New()
	if _, err := io.Copy(out, io.TeeReader(resp.Body, h)); err != nil {
		os.Remove(dest)
		return fmt.Errorf("写入失败 %s: %v", dest, err)
	}

	// 验证 SHA1
	if expectedSHA1 != "" {
		computed := hex.EncodeToString(h.Sum(nil))
		if computed != expectedSHA1 {
			os.Remove(dest)
			return fmt.Errorf("SHA1 不匹配: 期望 %s, 实际 %s", expectedSHA1, computed)
		}
	}
	return nil
}

func sha1OfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func unzip(archive, dest string, exclude []string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		// 防止 zip slip
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}
		// 检查排除列表
		skip := false
		for _, ex := range exclude {
			if strings.HasPrefix(f.Name, ex) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		ensureDir(filepath.Dir(path))
		out, err := os.Create(path)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
	}
	return nil
}

func untarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		path := filepath.Join(dest, header.Name)
		// 防止路径遍历
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(path, 0755)
		case tar.TypeReg:
			ensureDir(filepath.Dir(path))
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

// httpGet 发送 GET 请求
func httpGet(url string, headers ...string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", mirrorUrl(url, cfg.DownloadSource), nil)
	if err != nil {
		return nil, 0, err
	}
	for i := 0; i < len(headers)-1; i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	client := &http.Client{Timeout: 30 * 1000000000} // 30秒
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// httpPostJSON 发送 POST JSON 请求
func httpPostJSON(url, body string) ([]byte, int, error) {
	req, err := http.NewRequest("POST", mirrorUrl(url, cfg.DownloadSource), strings.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * 1000000000}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}
