package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"anyzone/internal/core/config"
)

type StoreData struct {
	Settings config.Settings  `json:"settings"`
	Accounts []config.Account `json:"accounts"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	data     StoreData
}

func NewStore() (*Store, error) {
	configPath, err := resolveConfigFilePath()
	if err != nil {
		return nil, err
	}

	s := &Store{
		filePath: configPath,
		data: StoreData{
			Settings: config.DefaultSettings(),
			Accounts: make([]config.Account, 0),
		},
	}

	if err := s.load(); err != nil {
		// 如果文件不存在，初始化空文件
		if os.IsNotExist(err) {
			if err := s.save(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return s, nil
}

// resolveConfigFilePath 自动判断便携模式或系统标准配置目录
func resolveConfigFilePath() (string, error) {
	// 检查当前目录下是否存在便携目录 anyzone_data
	portableDir := "anyzone_data"
	if fi, err := os.Stat(portableDir); err == nil && fi.IsDir() {
		return filepath.Join(portableDir, "anyzone.json"), nil
	}

	// 否则使用系统标准用户配置目录
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "."
	}

	targetDir := filepath.Join(userConfigDir, "anyzone")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(targetDir, "anyzone.json"), nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var data StoreData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	s.data = data
	return nil
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(s.filePath, raw, 0600)
}

func (s *Store) GetSettings() config.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings
}

func (s *Store) SaveSettings(settings config.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Settings = settings
	return s.save()
}

func (s *Store) ListAccounts() []config.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	accounts := make([]config.Account, len(s.data.Accounts))
	copy(accounts, s.data.Accounts)
	return accounts
}

func (s *Store) AddAccount(acc config.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Accounts = append(s.data.Accounts, acc)
	return s.save()
}

func (s *Store) UpdateAccount(acc config.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.data.Accounts {
		if a.ID == acc.ID {
			s.data.Accounts[i] = acc
			return s.save()
		}
	}
	return os.ErrNotExist
}

func (s *Store) DeleteAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]config.Account, 0, len(s.data.Accounts))
	for _, a := range s.data.Accounts {
		if a.ID != id {
			filtered = append(filtered, a)
		}
	}
	s.data.Accounts = filtered
	return s.save()
}
