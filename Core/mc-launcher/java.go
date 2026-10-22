package core

import (
	"fmt"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var Err = errors.New("")

type JavaInfo struct {
	Path    string
	Major   int
	Version string
}

func scanJava() []JavaInfo {
	var out []JavaInfo
	seen := map[string]bool{}
	add := func(p string) {
		if seen[p] {
			return
		}
		seen[p] = true
		ji := probeJava(p)
		if ji.Major > 0 {
			out = append(out, ji)
		}
	}
	if cfg.JavaPath != "" {
		add(cfg.JavaPath)
	}
	if v := os.Getenv("JAVA_HOME"); v != "" {
		add(filepath.Join(v, "bin", "java"))
		add(filepath.Join(v, "bin", "java.exe"))
	}
	javaDir := filepath.Join(baseDir, "java")
	if info, err := os.Stat(javaDir); err == nil && info.IsDir() {
		filepath.Walk(javaDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			b := filepath.Base(p)
			if b == "java" || b == "java.exe" {
				add(p)
			}
			return nil
		})
	}
	
	var javaCmd string
	if runtimeGOOS == "windows" {
		javaCmd = "where"
	} else {
	 javaCmd = "which"
	}
	if out, err := exec.Command(javaCmd, "java").Output(); err == nil {
		add(strings.TrimSpace(string(out)))
	}
	
	for _, root := range commonJavaRoots() {
		filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			b := filepath.Base(p)
			if b == "java" || b == "java.exe" {
				add(p)
			}
			return nil
		})
	}
	return out
}

func commonJavaRoots() []string {
	switch runtimeGOOS {
	case "windows":
		return []string{
			"C:\\Program Files\\Java",
			"C:\\Program Files (x86)\\Java",
			"C:\\Program Files\\Eclipse Adoptium",
		}
	case "darwin":
		return []string{
			"/Library/Java/JavaVirtualMachines",
			"/usr/local/lib/jvm",
			"/opt/homebrew/lib/jvm",
		}
	default:
		return []string{
			"/usr/lib/jvm",
			"/usr/java",
			"/opt/java",
			"/opt/jdk",
		}
	}
}

func probeJava(path string) JavaInfo {
	ji := JavaInfo{Path: path}
	if _, err := os.Stat(path); err != nil {
		return ji
	}
	cmd := exec.Command(path, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ji
	}
	ji.Version = string(out)
	
	re := regexp.MustCompile(`version "(?:1\.)?(\d+)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) > 1 {
		ji.Major = parseIntClamp(matches[1])
	}
	return ji
}

func pickJava(needMajor int) *JavaInfo {
	all := scanJava()
	if len(all) == 0 {
		return nil
	}
	for _, j := range all {
		if j.Major >= needMajor {
			return &j
		}
	}
	return &all[len(all)-1]
}

func parseIntClamp(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

func installJava(major int) (string, error) {
	ext := "tar.gz"
	archive := "temurin-" + itoa(major) + ".tar.gz"
	if runtimeGOOS == "windows" {
		ext = "zip"
		archive = "temurin-" + itoa(major) + ".zip"
	}
	
	arch := runtimeGOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "arm64" {
		arch = "aarch64"
	}
	
	url := "https://api.adoptium.net/v3/binary/latest/" + itoa(major) + "/ga/" +
		runtimeGOOS + "/" + arch + "/jre/hotspot/normal/eclipse"
	dest := filepath.Join(baseDir, "java", archive)
	if err := downloadFile(mirrorUrl(url, cfg.DownloadSource), dest, ""); err != nil {
		return "", fmt.Errorf("下载 Java 失败: %v", err)
	}
	target := filepath.Join(baseDir, "java", "temurin-"+itoa(major))
	if ext == "zip" {
		unzip(dest, target, nil)
	} else {
		untarGz(dest, target)
	}
	os.Remove(dest)
	if found := findJavaBinary(target); found != "" {
		cfg.JavaPath = found
		cfg.save()
		return found, nil
	}
	return "", fmt.Errorf("未找到 Java")
}

func findJavaBinary(dir string) string {
	cands := []string{"bin/java", "bin/java.exe", "jre/bin/java", "jre/bin/java.exe"}
	for _, c := range cands {
		p := filepath.Join(dir, c)
		if fileExists(p) {
			return p
		}
	}
	var found string
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Base(p) == "java" || filepath.Base(p) == "java.exe" {
			found = p
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

func ScanJava() []JavaInfo {
	return scanJava()
}

func InstallJava(major int) (string, error) {
	return installJava(major)
}
