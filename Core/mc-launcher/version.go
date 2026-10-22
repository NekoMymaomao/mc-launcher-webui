package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

func fetchManifest(source string) (*VersionManifest, error) {
	body, code, err := httpGet(mirrorUrl("https://launchermeta.mojang.com/mc/game/version_manifest_v2.json", source))
	if err != nil || code != 200 {
		return nil, fmt.Errorf("获取版本列表失败: %v", err)
	}
	var m VersionManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("解析版本列表失败: %v", err)
	}
	return &m, nil
}

func fetchVersionDetail(id, source string) (*VersionDetail, error) {
	manifest, err := fetchManifest(source)
	if err != nil {
		return nil, err
	}
	
	var versionURL string
	for _, v := range manifest.Versions {
		if v.ID == id {
			versionURL = v.URL
			break
		}
	}
	
	if versionURL == "" {
		versionURL = "https://launchermeta.mojang.com/mc/game/" + id + "/version_manifest_v2.json"
	}
	
	body, code, err := httpGet(mirrorUrl(versionURL, source))
	if err != nil || code != 200 {
		body, code, err = httpGet(mirrorUrl("https://launchermeta.mojang.com/mc/game/"+id+".json", source))
		if err != nil || code != 200 {
			return nil, fmt.Errorf("获取版本 %s 信息失败: %v", id, err)
		}
	}
	
	return parseVersionJSON(string(body))
}

func parseVersionJSON(raw string) (*VersionDetail, error) {
	var v VersionDetail
	var j map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &j); err != nil {
		return nil, err
	}

	v.ID = stringValue(j, "id")
	v.Type = stringValue(j, "type")
	v.MainClass = stringValue(j, "mainClass")
	v.Assets = stringValue(j, "assets")
	if v.Assets == "" {
		v.Assets = v.ID
	}

	if raw, ok := j["assetIndex"]; ok {
		var ai struct{ ID, URL, SHA1 string; Size int64 }
		json.Unmarshal(raw, &ai)
		v.AssetIndex = Artifact{URL: ai.URL, SHA1: ai.SHA1, Size: ai.Size}
		v.AssetIndexID = ai.ID
		if v.Assets == "" {
			v.Assets = ai.ID
		}
	}
	if raw, ok := j["downloads"]; ok {
		var dl struct{ Client struct{ URL, SHA1 string; Size int64 } }
		json.Unmarshal(raw, &dl)
		v.Client = Artifact{URL: dl.Client.URL, SHA1: dl.Client.SHA1, Size: dl.Client.Size}
	}
	if raw, ok := j["javaVersion"]; ok {
		var jv struct{ MajorVersion int }
		json.Unmarshal(raw, &jv)
		v.JavaMajor = jv.MajorVersion
		if v.JavaMajor <= 0 {
			v.JavaMajor = 8
		}
	}
	if raw, ok := j["minecraftArguments"]; ok {
		json.Unmarshal(raw, &v.LegacyArgs)
	}
	if raw, ok := j["logging"]; ok {
		var lg struct{ Client struct{ Argument string } }
		json.Unmarshal(raw, &lg)
		v.LogArgument = lg.Client.Argument
	}

	// libraries
	if raw, ok := j["libraries"]; ok {
		var libs []map[string]json.RawMessage
		json.Unmarshal(raw, &libs)
		plat := platformOS()
		for _, libj := range libs {
			name := stringValue(libj, "name")
			if name == "" {
				continue
			}
			lib := Library{Name: name, MavenBase: "https://libraries.minecraft.net/"}
			if raw, ok := libj["rules"]; ok {
				json.Unmarshal(raw, &lib.Rules)
			}
			if raw, ok := libj["downloads"]; ok {
				var dl struct {
					Artifact    Artifact
					Classifiers map[string]Artifact
				}
				json.Unmarshal(raw, &dl)
				lib.Downloads = &Downloads{Artifact: dl.Artifact, Classifiers: dl.Classifiers}
				if dl.Artifact.URL != "" {
					lib.ResolvedURL = mirrorUrl(dl.Artifact.URL, cfg.DownloadSource)
					lib.ResolvedPath = dl.Artifact.Path
					if lib.ResolvedPath == "" {
						lib.ResolvedPath = mavenPath(name)
						lib.ResolvedURL = mirrorUrl(lib.MavenBase+lib.ResolvedPath, cfg.DownloadSource)
					}
				}
				nativesKey := "natives-" + plat
				lib.Natives = map[string]string{plat: nativesKey}
				for k, a := range dl.Classifiers {
					if strings.Contains(k, plat) && a.URL != "" {
						lib.NativeURL = mirrorUrl(a.URL, cfg.DownloadSource)
						lib.NativePath = mavenPath(name + "-" + k)
						lib.IsNative = true
						break
					}
				}
			}
			if raw, ok := libj["extract"]; ok {
				var ex struct{ Exclude []string }
				json.Unmarshal(raw, &ex)
				lib.Extract = &Extract{Exclude: ex.Exclude}
			}
			lib.ShouldUse = ruleAllowed(lib.Rules)
			if lib.ShouldUse {
				v.Libraries = append(v.Libraries, lib)
			}
		}
	}

	// arguments
	if raw, ok := j["arguments"]; ok {
		var args struct{ Game, JVM []json.RawMessage }
		json.Unmarshal(raw, &args)
		parseArgs(args.Game, &v.GameArgs)
		parseArgs(args.JVM, &v.JVMArgs)
	}

	v.RawJSON = raw
	v.Valid = true
	return &v, nil
}

func parseArgs(raws []json.RawMessage, dst *[]string) {
	for _, raw := range raws {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			*dst = append(*dst, s)
			continue
		}
		var el struct{ Rules []Rule; Value json.RawMessage }
		if err := json.Unmarshal(raw, &el); err != nil || !ruleAllowed(el.Rules) {
			continue
		}
		if len(el.Value) == 0 {
			continue
		}
		// value 可能是单个字符串，也可能是字符串数组
		var one string
		if err := json.Unmarshal(el.Value, &one); err == nil {
			*dst = append(*dst, one)
			continue
		}
		var many []string
		if err := json.Unmarshal(el.Value, &many); err == nil {
			*dst = append(*dst, many...)
		}
	}
}

func mavenPath(name string) string {
	parts := strings.SplitN(name, ":", 4)
	if len(parts) < 3 {
		return name + ".jar"
	}
	group := strings.ReplaceAll(parts[0], ".", "/")
	art := parts[1]
	ver := parts[2]
	cls := ""
	if len(parts) > 3 {
		cls = "-" + parts[3]
	}
	return group + "/" + art + "/" + ver + "/" + art + "-" + ver + cls + ".jar"
}

func platformOS() string {
	switch runtimeGOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	default:
		return "linux"
	}
}

func ruleAllowed(rules []Rule) bool {
	if len(rules) == 0 {
		return true
	}
	allow := false
	for _, r := range rules {
		if r.matches() {
			allow = (r.Action == "allow")
		}
	}
	return allow
}

func (r Rule) matches() bool {
	if r.OS != nil {
		if r.OS.Name != "" && r.OS.Name != platformOS() {
			return false
		}
	}
	for k, v := range r.Features {
		switch k {
		case "is_demo_user":
			if v {
				return false
			}
		}
	}
	return true
}

func stringValue(m map[string]json.RawMessage, key string) string {
	b, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	json.Unmarshal(b, &s)
	return s
}
