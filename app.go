package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"anyzone/internal/core/config"
	"anyzone/internal/core/dnsutil"
	"anyzone/internal/core/security"
	"anyzone/internal/core/store"
	"anyzone/internal/provider"
)

// ZoneCheckReq 域名 NS 批量检测请求项
type ZoneCheckReq struct {
	AccountID string `json:"account_id,omitempty"`
	Domain    string `json:"domain"`
	Provider  string `json:"provider"`
}

type App struct {
	ctx        context.Context
	store      *store.Store
	sec        *security.SecurityManager
	mu         sync.RWMutex
	sessionKey []byte
	lastActive time.Time
}

func NewApp() (*App, error) {
	s, err := store.NewStore()
	if err != nil {
		return nil, fmt.Errorf("初始化本地数据仓库失败: %w", err)
	}

	return &App{
		store: s,
		sec:   security.NewSecurityManager(),
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) touchActivity() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastActive = time.Now()
}

// IsUnlocked 检查当前是否已解锁
func (a *App) IsUnlocked() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.sessionKey) > 0
}

// Lock 锁定应用，清空内存密钥
func (a *App) Lock() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionKey = nil
}

// IsPinConfigured 检查是否已经设置过 PIN
func (a *App) IsPinConfigured() bool {
	return a.store.GetSettings().IsPinConfigured
}

// SetupPin 初次设置 PIN 码
func (a *App) SetupPin(pin string) error {
	if len(pin) < 4 {
		return errors.New("PIN 码长度至少为 4 位")
	}

	salt, err := security.GenerateSalt()
	if err != nil {
		return err
	}

	key := security.DeriveKey(pin, salt)
	verifier, err := security.GenerateVerifier(key)
	if err != nil {
		return err
	}

	settings := a.store.GetSettings()
	settings.IsPinConfigured = true
	settings.PinLength = len(pin)
	settings.PinSalt = base64.StdEncoding.EncodeToString(salt)
	settings.PinVerifier = verifier

	if err := a.store.SaveSettings(settings); err != nil {
		return err
	}

	a.mu.Lock()
	a.sessionKey = key
	a.lastActive = time.Now()
	a.mu.Unlock()

	return nil
}

// VerifyPin 使用 PIN 码解锁
func (a *App) VerifyPin(pin string) error {
	settings := a.store.GetSettings()
	if !settings.IsPinConfigured {
		return security.ErrPINNotConfigured
	}

	salt, err := base64.StdEncoding.DecodeString(settings.PinSalt)
	if err != nil {
		return errors.New("解析安全凭证失败")
	}

	key, err := a.sec.VerifyPIN(pin, salt, settings.PinVerifier)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.sessionKey = key
	a.lastActive = time.Now()
	a.mu.Unlock()

	return nil
}

// ChangePin 修改 PIN 码
func (a *App) ChangePin(oldPin, newPin string) error {
	if len(newPin) < 4 {
		return errors.New("新 PIN 码长度至少为 4 位")
	}

	if err := a.VerifyPin(oldPin); err != nil {
		return fmt.Errorf("原 %w", err)
	}

	return a.SetupPin(newPin)
}

func (a *App) checkUnlockedKey() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.sessionKey) == 0 {
		return nil, errors.New("软件已锁定，请先输入 PIN 码解锁")
	}
	return a.sessionKey, nil
}

// GetSettings 读取配置
func (a *App) GetSettings() config.Settings {
	s := a.store.GetSettings()
	s.PinSalt = ""     // 脱敏
	s.PinVerifier = "" // 脱敏
	return s
}

// SaveSettings 更新配置
func (a *App) SaveSettings(newSettings config.Settings) error {
	oldSettings := a.store.GetSettings()
	oldSettings.AutoLockMinutes = newSettings.AutoLockMinutes
	oldSettings.ProxyEnabled = newSettings.ProxyEnabled
	oldSettings.ProxyURL = newSettings.ProxyURL
	oldSettings.ProxyRouting = newSettings.ProxyRouting
	oldSettings.Theme = newSettings.Theme
	oldSettings.AggregateAllDefault = newSettings.AggregateAllDefault
	if newSettings.ZonePageSize > 0 {
		oldSettings.ZonePageSize = newSettings.ZonePageSize
	}
	oldSettings.AutoHideMismatchedNS = newSettings.AutoHideMismatchedNS
	if newSettings.HiddenZones != nil {
		oldSettings.HiddenZones = newSettings.HiddenZones
	}
	return a.store.SaveSettings(oldSettings)
}

// ListAccounts 获取已保存的账号列表（密文不向前端透出明文凭据）
func (a *App) ListAccounts() ([]config.Account, error) {
	if _, err := a.checkUnlockedKey(); err != nil {
		return nil, err
	}
	a.touchActivity()

	accounts := a.store.ListAccounts()
	for i := range accounts {
		accounts[i].EncryptedCredentials = "" // 脱敏
	}
	return accounts, nil
}

// AddAccount 添加新厂商账号
func (a *App) AddAccount(name, providerType, credsJSON, customProxy string) (*config.Account, error) {
	key, err := a.checkUnlockedKey()
	if err != nil {
		return nil, err
	}
	a.touchActivity()

	encrypted, err := security.Encrypt(key, []byte(credsJSON))
	if err != nil {
		return nil, fmt.Errorf("凭据加密失败: %w", err)
	}

	acc := config.Account{
		ID:                   fmt.Sprintf("acc_%d", time.Now().UnixNano()),
		Name:                 name,
		Provider:             providerType,
		EncryptedCredentials: encrypted,
		CustomProxy:          customProxy,
		CreatedAt:            time.Now(),
	}

	if err := a.store.AddAccount(acc); err != nil {
		return nil, err
	}

	acc.EncryptedCredentials = ""
	return &acc, nil
}

// DeleteAccount 删除账号
func (a *App) DeleteAccount(id string) error {
	if _, err := a.checkUnlockedKey(); err != nil {
		return err
	}
	a.touchActivity()
	return a.store.DeleteAccount(id)
}

// GetAccountDetail 获取指定账号的解密详细配置供修改与回显
func (a *App) GetAccountDetail(accountID string) (map[string]interface{}, error) {
	key, err := a.checkUnlockedKey()
	if err != nil {
		return nil, err
	}
	a.touchActivity()

	accounts := a.store.ListAccounts()
	var target *config.Account
	for _, acc := range accounts {
		if acc.ID == accountID {
			target = &acc
			break
		}
	}
	if target == nil {
		return nil, errors.New("账号不存在")
	}

	credsData, err := security.Decrypt(key, target.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("凭据解密失败: %w", err)
	}

	var creds config.Credentials
	if err := json.Unmarshal(credsData, &creds); err != nil {
		return nil, fmt.Errorf("解析解密凭据失败: %w", err)
	}

	return map[string]interface{}{
		"id":           target.ID,
		"name":         target.Name,
		"provider":     target.Provider,
		"credentials":  creds,
		"custom_proxy": target.CustomProxy,
	}, nil
}

// UpdateAccount 修改现有云厂商账号配置
func (a *App) UpdateAccount(id, name, providerType, credsJSON, customProxy string) error {
	key, err := a.checkUnlockedKey()
	if err != nil {
		return err
	}
	a.touchActivity()

	accounts := a.store.ListAccounts()
	var target *config.Account
	for _, acc := range accounts {
		if acc.ID == id {
			target = &acc
			break
		}
	}
	if target == nil {
		return errors.New("账号不存在")
	}

	encrypted, err := security.Encrypt(key, []byte(credsJSON))
	if err != nil {
		return fmt.Errorf("凭据加密失败: %w", err)
	}

	updated := *target
	updated.Name = name
	updated.Provider = providerType
	updated.EncryptedCredentials = encrypted
	updated.CustomProxy = customProxy

	return a.store.UpdateAccount(updated)
}

// TestAccountConnection 测试凭据可用性
func (a *App) TestAccountConnection(providerType, credsJSON string) error {
	a.touchActivity()

	var creds config.Credentials
	if err := json.Unmarshal([]byte(credsJSON), &creds); err != nil {
		return fmt.Errorf("解析凭据参数格式失败: %w", err)
	}

	acc := config.Account{
		Provider: providerType,
	}

	settings := a.store.GetSettings()
	p, err := provider.CreateProvider(acc, creds, settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	return p.TestConnection(ctx)
}

func (a *App) getProviderForAccount(accountID string) (provider.DNSProvider, error) {
	key, err := a.checkUnlockedKey()
	if err != nil {
		return nil, err
	}

	accounts := a.store.ListAccounts()
	var target *config.Account
	for _, acc := range accounts {
		if acc.ID == accountID {
			target = &acc
			break
		}
	}
	if target == nil {
		return nil, errors.New("账号不存在")
	}

	credsData, err := security.Decrypt(key, target.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("凭据解密失败: %w", err)
	}

	var creds config.Credentials
	if err := json.Unmarshal(credsData, &creds); err != nil {
		return nil, fmt.Errorf("解析解密凭据失败: %w", err)
	}

	settings := a.store.GetSettings()
	return provider.CreateProvider(*target, creds, settings)
}

func isZoneHidden(accountID, zoneName string, hiddenList []string) bool {
	targetKey := fmt.Sprintf("%s:%s", accountID, zoneName)
	for _, h := range hiddenList {
		// 必须严格匹配具体账号ID与域名，避免跨厂商误伤
		if h == targetKey {
			return true
		}
	}
	return false
}

// ListZones 获取指定账号下的全部域名
func (a *App) ListZones(accountID string) ([]provider.Zone, error) {
	a.touchActivity()
	p, err := a.getProviderForAccount(accountID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	zones, err := p.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	settings := a.store.GetSettings()
	var accName string
	var accProvider string
	for _, acc := range a.store.ListAccounts() {
		if acc.ID == accountID {
			accName = acc.Name
			accProvider = acc.Provider
			break
		}
	}

	for i := range zones {
		zones[i].AccountID = accountID
		zones[i].AccountName = accName
		if zones[i].Provider == "" {
			zones[i].Provider = accProvider
		}
		zones[i].IsHidden = isZoneHidden(accountID, zones[i].Name, settings.HiddenZones)
	}

	return zones, nil
}

// ListAllZones 全选模式：聚合拉取所有云厂商账号下的域名
func (a *App) ListAllZones() ([]provider.Zone, error) {
	if _, err := a.checkUnlockedKey(); err != nil {
		return nil, err
	}
	a.touchActivity()

	accounts := a.store.ListAccounts()
	if len(accounts) == 0 {
		return []provider.Zone{}, nil
	}

	settings := a.store.GetSettings()
	var allZones []provider.Zone
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发拉取各个账号域名，上限控制在 5 个并发拉取任务
	sem := make(chan struct{}, 5)

	for _, acc := range accounts {
		wg.Add(1)
		go func(targetAcc config.Account) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			p, err := a.getProviderForAccount(targetAcc.ID)
			if err != nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()

			zones, err := p.ListZones(ctx)
			if err != nil {
				return
			}

			for i := range zones {
				zones[i].AccountID = targetAcc.ID
				zones[i].AccountName = targetAcc.Name
				if zones[i].Provider == "" {
					zones[i].Provider = targetAcc.Provider
				}
				zones[i].IsHidden = isZoneHidden(targetAcc.ID, zones[i].Name, settings.HiddenZones)
			}

			mu.Lock()
			allZones = append(allZones, zones...)
			mu.Unlock()
		}(acc)
	}

	wg.Wait()
	// 返回完整拉取的各账号全部域名，不再截断，由前端在侧边栏做丝滑高效的分页渲染
	return allZones, nil
}

// CheckZoneNS 检测单一域名的权威 NS 及厂商匹配
func (a *App) CheckZoneNS(domain string, expectedProvider string) dnsutil.NSCheckResult {
	a.touchActivity()
	return dnsutil.CheckDomainNS(context.Background(), domain, expectedProvider)
}

// CheckZonesNSBatch 并发批量检测多个域名的权威 NS 归属
func (a *App) CheckZonesNSBatch(items []ZoneCheckReq) []dnsutil.NSCheckResult {
	a.touchActivity()
	results := make([]dnsutil.NSCheckResult, len(items))
	if len(items) == 0 {
		return results
	}

	var wg sync.WaitGroup
	// 限制并发 worker 数量为 8
	sem := make(chan struct{}, 8)

	for i, req := range items {
		wg.Add(1)
		go func(idx int, r ZoneCheckReq) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res := dnsutil.CheckDomainNS(ctx, r.Domain, r.Provider)
			res.AccountID = r.AccountID
			results[idx] = res
		}(i, req)
	}

	wg.Wait()
	return results
}

// ToggleHideZone 切换指定域名的隐藏状态（严格以账号 ID 隔离）
func (a *App) ToggleHideZone(accountID, zoneName string, hide bool) error {
	if _, err := a.checkUnlockedKey(); err != nil {
		return err
	}
	a.touchActivity()

	settings := a.store.GetSettings()
	targetKey := fmt.Sprintf("%s:%s", accountID, zoneName)

	var updated []string
	for _, h := range settings.HiddenZones {
		// 严格过滤匹配当前账号的 key，并在解除隐藏时清理可能残留的旧版未带前缀记录
		if h != targetKey && (hide || h != zoneName) {
			updated = append(updated, h)
		}
	}

	if hide {
		updated = append(updated, targetKey)
	}

	settings.HiddenZones = updated
	return a.store.SaveSettings(settings)
}

// ListRecords 获取指定域名的全部解析记录
func (a *App) ListRecords(accountID, zoneID, zoneName string) ([]provider.Record, error) {
	a.touchActivity()
	p, err := a.getProviderForAccount(accountID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	return p.ListRecords(ctx, zoneID, zoneName)
}

// CreateRecord 新增一条解析记录
func (a *App) CreateRecord(accountID, zoneID, zoneName string, r provider.Record) (*provider.Record, error) {
	a.touchActivity()
	p, err := a.getProviderForAccount(accountID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return p.CreateRecord(ctx, zoneID, zoneName, r)
}

// UpdateRecord 修改一条解析记录
func (a *App) UpdateRecord(accountID, zoneID, zoneName string, r provider.Record) (*provider.Record, error) {
	a.touchActivity()
	p, err := a.getProviderForAccount(accountID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return p.UpdateRecord(ctx, zoneID, zoneName, r)
}

// DeleteRecord 删除一条解析记录
func (a *App) DeleteRecord(accountID, zoneID, zoneName, recordID string) error {
	a.touchActivity()
	p, err := a.getProviderForAccount(accountID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return p.DeleteRecord(ctx, zoneID, zoneName, recordID)
}
