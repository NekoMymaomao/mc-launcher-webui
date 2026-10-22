package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

func getInstalledDir(versionId string) string {
	if cfg.VersionIsolation {
		return filepath.Join(baseDir, "versions", versionId)
	}
	return filepath.Join(baseDir, ".minecraft", "versions", versionId)
}

func getMinecraftDir() string {
	if cfg.VersionIsolation {
		return filepath.Join(baseDir, "versions")
	}
	return filepath.Join(baseDir, ".minecraft")
}

func getLibDir(versionId string) string {
	if cfg.VersionIsolation {
		return filepath.Join(baseDir, "libraries")
	}
	return filepath.Join(getMinecraftDir(), "versions", versionId, "libraries")
}

func installVersion(versionId string, cb string, sender func(Progress)) (*Response, error) {
	detail, err := fetchVersionDetail(versionId, cfg.DownloadSource)
	if err != nil {
		return &Response{Ok: false, Error: "获取版本信息失败: " + err.Error()}, nil
	}
	if !detail.Valid {
		return &Response{Ok: false, Error: "版本不存在: " + versionId}, nil
	}

	instDir := getInstalledDir(versionId)
	libDir := getLibDir(versionId)
	ensureDir(instDir)
	ensureDir(libDir)
	ensureDir(filepath.Join(baseDir, "assets", "indexes"))
	ensureDir(filepath.Join(baseDir, "assets", "objects"))

	// 保存版本 manifest（优先写入原始 JSON，保证可被其他启动器识别）
	manifestText := detail.RawJSON
	if manifestText == "" {
		b, _ := json.MarshalIndent(detail, "", "  ")
		manifestText = string(b)
	}
	writeAll(filepath.Join(instDir, versionId+".json"), manifestText)

	var wg sync.WaitGroup
	var mu sync.Mutex
	downloaded := 0
	totalTasks := 0

	totalTasks++
	totalTasks++
	for _, lib := range detail.Libraries {
		if lib.ResolvedURL != "" {
			totalTasks++
		}
		if lib.NativeURL != "" {
			totalTasks++
		}
	}

	updateProgress := func(msg string) {
		mu.Lock()
		d := downloaded
		mu.Unlock()
		if cb != "" && sender != nil {
			sender(Progress{ID: cb, Done: int64(d), Total: int64(totalTasks), Msg: msg})
		}
	}

	// 1. 下载 client jar
	wg.Add(1)
	go func() {
		defer wg.Done()
		dest := filepath.Join(instDir, versionId+".jar")
		if err := downloadFile(detail.Client.URL, dest, detail.Client.SHA1, nil); err != nil {
			logError("下载客户端失败: %v", err)
		}
		mu.Lock(); downloaded++; mu.Unlock()
		updateProgress("下载客户端...")
	}()

	// 2. 下载 libraries
	sem := make(chan struct{}, cfg.Threads)
	for _, lib := range detail.Libraries {
		if lib.ResolvedURL != "" {
			sem <- struct{}{}
			wg.Add(1)
			go func(l Library) {
				defer func() { <-sem; wg.Done() }()
				dest := filepath.Join(libDir, l.ResolvedPath)
				if err := downloadFile(l.ResolvedURL, dest, l.Downloads.Artifact.SHA1, nil); err != nil {
					logError("下载库失败 %s: %v", l.Name, err)
				}
				mu.Lock(); downloaded++; mu.Unlock()
				updateProgress("下载库: " + l.Name)
			}(lib)
		}
		if lib.NativeURL != "" {
			sem <- struct{}{}
			wg.Add(1)
			go func(l Library) {
				defer func() { <-sem; wg.Done() }()
				dest := filepath.Join(libDir, l.NativePath)
				if err := downloadFile(l.NativeURL, dest, "", nil); err != nil {
					logError("下载 natives 失败 %s: %v", l.Name, err)
				} else if l.Extract != nil {
					nativesDir := filepath.Join(instDir, "natives")
					exclude := []string{"META-INF/"}
					if l.Extract.Exclude != nil {
						exclude = l.Extract.Exclude
					}
					unzip(dest, nativesDir, exclude)
				}
				mu.Lock(); downloaded++; mu.Unlock()
				updateProgress("下载 natives: " + l.Name)
			}(lib)
		}
	}

	// 3. 下载 asset index
	wg.Add(1)
	go func() {
		defer wg.Done()
		if detail.AssetIndex.URL != "" {
			dest := filepath.Join(baseDir, "assets", "indexes", detail.AssetIndexID+".json")
			if err := downloadFile(detail.AssetIndex.URL, dest, detail.AssetIndex.SHA1, nil); err != nil {
				logError("下载 asset index 失败: %v", err)
			}
		}
		mu.Lock(); downloaded++; mu.Unlock()
		updateProgress("下载资源索引...")
	}()

	wg.Wait()
	updateProgress("安装完成!")
	return &Response{Ok: true, Data: map[string]string{"id": versionId, "path": instDir}}, nil
}

func listVersions() []string {
	var out []string
	checkDir := func(dir string) {
		dirs, err := listDirs(dir)
		if err != nil {
			return
		}
		for _, d := range dirs {
			if fileExists(filepath.Join(dir, d, d+".json")) {
				out = append(out, d)
			}
		}
	}
	checkDir(filepath.Join(baseDir, "versions"))
	checkDir(filepath.Join(baseDir, ".minecraft", "versions"))
	return out
}

func removeVersion(versionId string) error {
	for _, base := range []string{
		filepath.Join(baseDir, "versions", versionId),
		filepath.Join(baseDir, ".minecraft", "versions", versionId),
	} {
		if info, err := os.Stat(base); err == nil && info.IsDir() {
			os.RemoveAll(base)
		}
	}
	return nil
}

func verifyVersion(versionId string) (bool, string) {
	detail, err := fetchVersionDetail(versionId, cfg.DownloadSource)
	if err != nil {
		return false, err.Error()
	}
	if !detail.Valid {
		return false, "版本信息无效"
	}
	instDir := getInstalledDir(versionId)
	clientJar := filepath.Join(instDir, versionId+".jar")
	if !fileExists(clientJar) {
		return false, "客户端 jar 缺失: " + clientJar
	}
	if detail.Client.SHA1 != "" {
		if sha, err := sha1OfFile(clientJar); err != nil || sha != detail.Client.SHA1 {
			return false, "客户端 jar SHA1 不匹配"
		}
	}
	return true, "验证通过"
}

func ListVersions() []string {
	return listVersions()
}

func InstallVersion(versionId, cb string, sender func(Progress)) (*Response, error) {
	return installVersion(versionId, cb, sender)
}

func RemoveVersion(versionId string) error {
	return removeVersion(versionId)
}

func VerifyVersion(versionId string) (bool, string) {
	return verifyVersion(versionId)
}
