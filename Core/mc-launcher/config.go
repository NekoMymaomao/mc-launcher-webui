package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func settingsPath() string {
	return filepath.Join(baseDir, "settings.json")
}

func accountsPath() string {
	return filepath.Join(baseDir, "accounts.json")
}

func loadSettings() *Settings {
	s := &Settings{
		MaxMemory:        2048,
		MinMemory:        1024,
		DownloadSource:   "official",
		Width:            854,
		Height:           480,
		Threads:          8,
		VersionIsolation: false,
	}
	b, err := os.ReadFile(settingsPath())
	if err == nil {
		json.Unmarshal(b, s)
	}
	if s.MaxMemory < 256 {
		s.MaxMemory = 2048
	}
	if s.MinMemory < 256 || s.MinMemory > s.MaxMemory {
		s.MinMemory = 1024
	}
	if s.Threads <= 0 {
		s.Threads = 8
	}
	return s
}

func (s *Settings) save() error {
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(settingsPath(), b, 0644)
}

func loadAccounts() *AccountStore {
	as := &AccountStore{}
	b, err := os.ReadFile(accountsPath())
	if err == nil {
		json.Unmarshal(b, as)
	}
	return as
}

func saveAccounts() {
	if accounts == nil {
		return
	}
	b, _ := json.MarshalIndent(accounts, "", "  ")
	os.WriteFile(accountsPath(), b, 0644)
}
