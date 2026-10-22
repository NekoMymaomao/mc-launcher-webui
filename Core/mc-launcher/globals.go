package main

import "os"

var (
	baseDir  string
	cfg      *Settings
	accounts *AccountStore
	curLang  string
)

func init() {
	curLang = "en"
	if v := os.Getenv("MC_LAUNCHER_LANG"); v == "zh" || v == "zh-CN" {
		curLang = "zh"
	}
}
