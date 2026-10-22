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
	assetsDir := filepath.Join(baseDir, "assets")
	ensureDir(instDir)
	ensureDir(libDir)
	ensureDir(filepath.Join(instDir, "natives"))
	ensureDir(filepath.Join(assetsDir, "indexes"))
	ensureDir(filepath.Join(assetsDir, "objects"))

	// 保存版本 manifest（优先写入原始 JSON，保证可被其他启动器识别）
	manifestText := detail.RawJSON
	if manifestText == "" {
		b, _ := json.MarshalIndent(detail, "", "  ")
		manifestText = string(b)
	}
	if err := writeAll(filepath.Join(instDir, versionId+".json"), manifestText); err != nil {
		return &Response{Ok: false, Error: "写入版本文件失败: " + err.Error()}, nil
	}

	var mu sync.Mutex
	done, total := 0, 1
	var errs []string
	report := func(msg string) {
		mu.Lock()
		d, t := done, total
		mu.Unlock()
		if cb != "" && sender != nil {
			sender(Progress{ID: cb, Done: int64(d), Total: int64(t), Msg: msg})
		}
	}
	mark := func() {
		mu.Lock()
		done++
		mu.Unlock()
	}
	recordErr := func(msg string) {
		mu.Lock()
		errs = append(errs, msg)
		mu.Unlock()
	}

	threads := cfg.Threads
	if threads <= 0 {
		threads = 8
	}
	sem := make(chan struct{}, threads)
	run := func(fn func()) {
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			fn()
		}()
	}

	// ---------- 阶段 1: client jar + libraries + natives ----------
	var wg sync.WaitGroup
	wg.Add(1)
	run(func() {
		defer wg.Done()
		dest := filepath.Join(instDir, versionId+".jar")
		if err := downloadFile(detail.Client.URL, dest, detail.Client.SHA1); err != nil {
			recordErr("客户端: " + err.Error())
			logError("下载客户端失败: %v", err)
		}
		mark()
		report("下载客户端...")
	})

	for _, lib := range detail.Libraries {
		l := lib
		if l.ResolvedURL != "" {
			wg.Add(1)
			run(func() {
				defer wg.Done()
				dest := filepath.Join(libDir, l.ResolvedPath)
				sha := ""
				if l.Downloads != nil {
					sha = l.Downloads.Artifact.SHA1
				}
				if err := downloadFile(l.ResolvedURL, dest, sha); err != nil {
					recordErr(l.Name + ": " + err.Error())
					logError("下载库失败 %s: %v", l.Name, err)
				}
				mark()
				report("下载库: " + l.Name)
			})
		}
		if l.NativeURL != "" {
			wg.Add(1)
			run(func() {
				defer wg.Done()
				dest := filepath.Join(libDir, l.NativePath)
				sha := ""
				if l.Downloads != nil {
					for _, a := range l.Downloads.Classifiers {
						if a.Path == l.NativePath {
							sha = a.SHA1
							break
						}
					}
				}
				if err := downloadFile(l.NativeURL, dest, sha); err != nil {
					recordErr("natives " + l.Name + ": " + err.Error())
					logError("下载 natives 失败 %s: %v", l.Name, err)
				} else if l.Extract != nil {
					exclude := []string{"META-INF/"}
					if len(l.Extract.Exclude) > 0 {
						exclude = l.Extract.Exclude
					}
					if err := unzip(dest, filepath.Join(instDir, "natives"), exclude); err != nil {
						recordErr("解压 natives " + l.Name + ": " + err.Error())
					}
				}
				mark()
				report("下载 natives: " + l.Name)
			})
		}
	}
	wg.Wait()

	// ---------- 阶段 2: asset index（必须在 objects 之前完成）----------
	if detail.AssetIndex.URL != "" && detail.AssetIndexID != "" {
		indexPath := filepath.Join(assetsDir, "indexes", detail.AssetIndexID+".json")
		if err := downloadFile(detail.AssetIndex.URL, indexPath, detail.AssetIndex.SHA1); err != nil {
			recordErr("资源索引: " + err.Error())
			logError("下载 asset index 失败: %v", err)
		} else {
			report("下载资源索引...")
		}

		// ---------- 阶段 3: asset objects ----------
		raw, rerr := os.ReadFile(indexPath)
		if rerr != nil {
			recordErr("读取资源索引: " + rerr.Error())
		} else {
			var idx struct {
				Objects map[string]struct {
					Hash string `json:"hash"`
					Size int64  `json:"size"`
				} `json:"objects"`
			}
			if jerr := json.Unmarshal(raw, &idx); jerr != nil {
				recordErr("解析资源索引: " + jerr.Error())
			} else {
				mu.Lock()
				total += len(idx.Objects)
				mu.Unlock()

				var awg sync.WaitGroup
				for hash := range idx.Objects {
					h := hash
					if len(h) < 4 {
						mark()
						continue
					}
					awg.Add(1)
					run(func() {
						defer awg.Done()
						sub := filepath.Join(assetsDir, "objects", h[:2])
						dest := filepath.Join(sub, h)
						url := "https://resources.download.minecraft.net/" + h[:2] + "/" + h
						if err := downloadFile(url, dest, h); err != nil {
							recordErr("资源 " + h[:8] + ": " + err.Error())
						}
						mark()
					})
				}
				awg.Wait()
				report("下载资源完成")
			}
		}
	}

	updateProgressFinal := func() {
		mu.Lock()
		d, t := done, total
		n := len(errs)
		var first string
		if n > 0 {
			first = errs[0]
		}
		mu.Unlock()
		if cb != "" && sender != nil {
			sender(Progress{ID: cb, Done: int64(d), Total: int64(t), Msg: "安装完成"})
		}
		if n > 0 {
			logError("安装完成，但有 %d 个文件失败（首个：%s）", n, first)
		}
	}
	updateProgressFinal()

	if len(errs) > 0 {
		return &Response{Ok: false, Error: "安装完成但部分文件失败（" + itoaInt(len(errs)) + " 个）: " + errs[0]}, nil
	}
	return &Response{Ok: true, Data: map[string]string{"id": versionId, "path": instDir}}, nil
}

func itoaInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
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
