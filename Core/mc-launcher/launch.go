package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func launchVersion(versionId, username string) (*Response, error) {
	var accessToken, uuid, userType string
	if acc := accounts.current(); acc != nil {
		accessToken = acc.AccessToken
		uuid = acc.UUID
		userType = acc.Type
	}

	if accessToken == "" {
		acc := loginOffline(username)
		accessToken = acc.AccessToken
		uuid = acc.UUID
		userType = "offline"
	}

	detail, err := fetchVersionDetail(versionId, cfg.DownloadSource)
	if err != nil {
		return &Response{Ok: false, Error: "获取版本信息失败: " + err.Error()}, nil
	}

	instDir := getInstalledDir(versionId)
	nativesDir := filepath.Join(instDir, "natives")
	ensureDir(nativesDir)

	needJava := detail.JavaMajor
	if needJava <= 0 {
		needJava = 8
	}
	java := pickJava(needJava)
	if java == nil {
		return &Response{Ok: false, Error: "未找到 Java " + itoa(needJava) + "+，请先安装 Java"}, nil
	}

	var cp []string
	cp = append(cp, filepath.Join(instDir, versionId+".jar"))
	for _, lib := range detail.Libraries {
		if lib.IsNative || lib.ResolvedPath == "" {
			continue
		}
		cp = append(cp, filepath.Join(getLibDir(versionId), lib.ResolvedPath))
	}

	sep := ":"
	if runtimeGOOS == "windows" {
		sep = ";"
	}
	cpStr := strings.Join(cp, sep)

	var allArgs []string
	for _, a := range detail.JVMArgs {
		allArgs = append(allArgs, substituteArgs(a, versionId, username, uuid, accessToken, instDir, nativesDir, cpStr, userType, detail.AssetIndexID))
	}
	allArgs = append(allArgs, "-Xmx"+itoa(cfg.MaxMemory)+"M")
	allArgs = append(allArgs, "-Xms"+itoa(cfg.MinMemory)+"M")
	for _, a := range cfg.JVMArgs {
		allArgs = append(allArgs, a)
	}
	allArgs = append(allArgs, "-cp", cpStr)
	allArgs = append(allArgs, detail.MainClass)

	for _, a := range detail.GameArgs {
		allArgs = append(allArgs, substituteArgs(a, versionId, username, uuid, accessToken, instDir, nativesDir, cpStr, userType, detail.AssetIndexID))
	}

	if detail.LegacyArgs != "" {
		for _, a := range strings.Fields(detail.LegacyArgs) {
			allArgs = append(allArgs, substituteArgs(a, versionId, username, uuid, accessToken, instDir, nativesDir, cpStr, userType, detail.AssetIndexID))
		}
	}

	allArgs = append(allArgs, "--username", username)
	allArgs = append(allArgs, "--version", versionId)
	allArgs = append(allArgs, "--gameDir", instDir)
	allArgs = append(allArgs, "--assetsDir", filepath.Join(baseDir, "assets"))
	allArgs = append(allArgs, "--assetIndex", detail.AssetIndexID)
	allArgs = append(allArgs, "--uuid", uuid)
	allArgs = append(allArgs, "--accessToken", accessToken)
	allArgs = append(allArgs, "--userType", userType)
	if detail.LogArgument != "" {
		allArgs = append(allArgs, detail.LogArgument)
	}

	cmd := exec.Command(java.Path, allArgs...)
	cmd.Dir = instDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logInfo("启动命令: %s", strings.Join(cmd.Args, " "))
	err = cmd.Run()
	if err != nil {
		return &Response{Ok: false, Error: "游戏启动失败: " + err.Error()}, nil
	}
	return &Response{Ok: true}, nil
}

func substituteArgs(arg, versionId, username, uuid, token, gameDir, nativesDir, cp, userType, assetIndex string) string {
	repls := map[string]string{
		"${auth_player_name}":     username,
		"${version_name}":         versionId,
		"${game_directory}":       gameDir,
		"${assets_root}":          filepath.Join(baseDir, "assets"),
		"${assets_index_name}":    assetIndex,
		"${auth_uuid}":            uuid,
		"${auth_access_token}":    token,
		"${user_type}":            userType,
		"${resolution_width}":     itoa(cfg.Width),
		"${resolution_height}":    itoa(cfg.Height),
		"${natives_directory}":    nativesDir,
		"${classpath}":            cp,
	}
	for k, v := range repls {
		arg = strings.ReplaceAll(arg, k, v)
	}
	return arg
}

func LaunchVersion(versionId, username string) (*Response, error) {
	return launchVersion(versionId, username)
}
