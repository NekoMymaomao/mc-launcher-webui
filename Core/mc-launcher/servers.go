package core

// servers.go — 服务器列表（servers.json）
import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ServerEntry 服务器列表项
type ServerEntry struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func serversPath() string {
	return filepath.Join(GetBaseDir(), "servers.json")
}

// LoadServers 读取服务器列表
func LoadServers() []ServerEntry {
	b, err := os.ReadFile(serversPath())
	if err != nil {
		return []ServerEntry{}
	}
	var list []ServerEntry
	if err := json.Unmarshal(b, &list); err != nil {
		return []ServerEntry{}
	}
	if list == nil {
		return []ServerEntry{}
	}
	return list
}

// SaveServers 写入服务器列表
func SaveServers(list []ServerEntry) error {
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(serversPath(), b, 0644)
}

// AddServer 添加或更新服务器
func AddServer(address, name string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return os.ErrInvalid
	}
	if name == "" {
		name = address
	}
	list := LoadServers()
	for i := range list {
		if list[i].Address == address {
			list[i].Name = name
			return SaveServers(list)
		}
	}
	list = append(list, ServerEntry{Name: name, Address: address})
	return SaveServers(list)
}

// RemoveServer 按地址删除服务器
func RemoveServer(address string) (bool, error) {
	list := LoadServers()
	out := list[:0]
	found := false
	for _, s := range list {
		if s.Address == address {
			found = true
			continue
		}
		out = append(out, s)
	}
	if !found {
		return false, nil
	}
	return true, SaveServers(out)
}
