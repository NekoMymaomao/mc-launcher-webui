package core

// api.go — 对外导出的库接口
import (
	"os"
	"path/filepath"
)

// Init 初始化核心模块（创建目录、加载配置与账号）
func Init() error {
	if baseDir == "" {
		baseDir = launcherDir()
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}
	for _, d := range []string{
		"versions", "libraries", "assets/indexes", "assets/objects",
		"java", "logs", "saves", "mods", "shaderpacks", "resourcepacks",
	} {
		os.MkdirAll(filepath.Join(baseDir, d), 0755)
	}
	cfg = loadSettings()
	if cfg.Lang != "" {
		curLang = cfg.Lang
	}
	accounts = loadAccounts()
	migrateAccounts()
	return nil
}

// InitWithDir 使用指定目录初始化
func InitWithDir(dir string) error {
	baseDir = dir
	return Init()
}

// GetBaseDir 返回数据目录
func GetBaseDir() string {
	if baseDir == "" {
		baseDir = launcherDir()
	}
	return baseDir
}

// ---------- 配置 ----------

// GetSettings 返回当前配置（只读使用）
func GetSettings() *Settings {
	if cfg == nil {
		cfg = loadSettings()
	}
	return cfg
}

// SaveSettings 更新并持久化配置
func SaveSettings(s *Settings) error {
	if cfg == nil {
		cfg = &Settings{}
	}
	if s.MaxMemory > 0 {
		cfg.MaxMemory = s.MaxMemory
	}
	if s.MinMemory > 0 {
		cfg.MinMemory = s.MinMemory
	}
	if s.Threads > 0 {
		cfg.Threads = s.Threads
	}
	if s.Width > 0 {
		cfg.Width = s.Width
	}
	if s.Height > 0 {
		cfg.Height = s.Height
	}
	if s.JavaPath != "" {
		cfg.JavaPath = s.JavaPath
	}
	if s.DownloadSource != "" {
		cfg.DownloadSource = s.DownloadSource
	}
	if s.ListenAddr != "" {
		cfg.ListenAddr = s.ListenAddr
	}
	if s.JVMArgs != nil {
		cfg.JVMArgs = s.JVMArgs
	}
	if s.Lang != "" {
		cfg.Lang = s.Lang
		curLang = s.Lang
	}
	if s.GameArgs != nil {
		cfg.GameArgs = s.GameArgs
	}
	return cfg.save()
}

// SetJavaPath 设置 Java 路径
func SetJavaPath(p string) error {
	if cfg == nil {
		cfg = loadSettings()
	}
	cfg.JavaPath = p
	return cfg.save()
}

// ---------- 账号 ----------

// ListAccounts 返回所有账号
func ListAccounts() []Account {
	if accounts == nil {
		accounts = loadAccounts()
	}
	return accounts.Accounts
}

// CurrentAccount 返回当前选中的账号，未登录返回 nil
func CurrentAccount() *Account {
	if accounts == nil {
		accounts = loadAccounts()
	}
	return accounts.current()
}

// UpsertAccount 新增或更新账号
func UpsertAccount(acc Account) error {
	if accounts == nil {
		accounts = loadAccounts()
	}
	accounts.upsert(acc)
	return nil
}

// RemoveAccount 按 UUID 删除账号
func RemoveAccount(uuid string) bool {
	if accounts == nil {
		accounts = loadAccounts()
	}
	return accounts.remove(uuid)
}

// SelectAccount 选中指定 UUID 的账号
func SelectAccount(uuid string) bool {
	if accounts == nil {
		accounts = loadAccounts()
	}
	for _, a := range accounts.Accounts {
		if a.UUID == uuid {
			accounts.Selected = uuid
			saveAccounts()
			return true
		}
	}
	return false
}

// migrateAccounts 修正旧版本按 Type 去重导致的 selected 不匹配
func migrateAccounts() {
	if accounts == nil || len(accounts.Accounts) == 0 {
		return
	}
	for _, a := range accounts.Accounts {
		if a.UUID == accounts.Selected {
			return
		}
	}
	accounts.Selected = accounts.Accounts[0].UUID
	saveAccounts()
}

// ---------- 语言 ----------

// GetLang 返回当前语言
func GetLang() string {
	if curLang == "" {
		return "en"
	}
	return curLang
}

// SetLang 设置语言并持久化到 settings.json
func SetLang(l string) {
	if l == "zh-CN" {
		l = "zh"
	}
	curLang = l
	if cfg == nil {
		return
	}
	cfg.Lang = l
	cfg.save()
}
