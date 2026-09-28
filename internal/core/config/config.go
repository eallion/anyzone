package config

import (
	"time"
)

// Provider 类型常量
const (
	// 国内主流公有云
	ProviderCloudflare      = "cloudflare"
	ProviderAliyun          = "aliyun"
	ProviderAliyunESA       = "aliyun_esa"
	ProviderDNSPod          = "dnspod"
	ProviderTencentCloud    = "tencent_cloud"
	ProviderTencentEdgeOne  = "tencent_edgeone"
	ProviderHuawei          = "huawei"
	ProviderVolcengine      = "volcengine"
	ProviderBaidu           = "baidu"

	// 国际云巨头
	ProviderAWSRoute53      = "aws_route53"
	ProviderGoogleCloud     = "gcp_dns"
	ProviderAzureDNS        = "azure_dns"
	ProviderOracleCloud     = "oci_dns"

	// 海外知名注册商与专业 DNS
	ProviderPorkbun         = "porkbun"
	ProviderHetzner         = "hetzner"
	ProviderDigitalOcean    = "digitalocean"
	ProviderVultr           = "vultr"
	ProviderLinode          = "linode"
	ProviderGoDaddy         = "godaddy"
	ProviderNamecheap       = "namecheap"
	ProviderDNSimple        = "dnsimple"
	ProviderBunny           = "bunny"
	ProviderDeSEC           = "desec"

	// 私有与自建 DNS
	ProviderPowerDNS        = "powerdns"
	ProviderBIND9           = "bind9_rfc2136"
	ProviderAdGuard         = "adguard_home"
)

// 代理路由模式
const (
	ProxyRoutingSmart = "smart" // 智能分流：海外走代理，国内直连
	ProxyRoutingAll   = "all"   // 全局代理
)

// 主题模式
const (
	ThemeAuto  = "auto"
	ThemeLight = "light"
	ThemeDark  = "dark" // Dark mode / 暗黑模式
)

// Account 代表一个 DNS 云厂商账号
type Account struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Provider             string    `json:"provider"`
	EncryptedCredentials string    `json:"encrypted_credentials"` // AES-256 加密后的凭据 JSON 字符串
	CustomProxy          string    `json:"custom_proxy,omitempty"` // 单账号特定代理（可选）
	CreatedAt            time.Time `json:"created_at"`
}

// Credentials 解密后的明文凭据结构
type Credentials struct {
	// 通用 Token (Cloudflare / DigitalOcean / Hetzner / Vultr / Linode / Bunny / deSEC / 华为云)
	ApiToken string `json:"api_token,omitempty"`

	// Key + Secret 模式 (阿里云 / 阿里云ESA / 腾讯云CAM / EdgeOne / 火山引擎 / AWS / GoDaddy / 百度云)
	AccessKeyId     string `json:"access_key_id,omitempty"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	ApiKey          string `json:"api_key,omitempty"`
	ApiSecret       string `json:"api_secret,omitempty"`
	SecretApiKey    string `json:"secret_api_key,omitempty"`

	// 腾讯云 / DNSPod 经典 Token
	TokenId string `json:"token_id,omitempty"`
	Token   string `json:"token,omitempty"`

	// 自建服务器 / 私有 DNS (PowerDNS / BIND 9 / AdGuard)
	ServerUrl string `json:"server_url,omitempty"`
	ZoneID    string `json:"zone_id,omitempty"`

	// 附属边缘服务开关 (共用主凭据)
	EnableESA     bool `json:"enable_esa,omitempty"`     // 阿里云：同时管理 ESA 边缘安全加速站点与解析
	EnableEdgeOne bool `json:"enable_edgeone,omitempty"` // 腾讯云：同时管理 EdgeOne 边缘安全加速站点与解析

	// 国际云与高阶特定配置
	Region    string `json:"region,omitempty"`     // AWS / 华为云 / 腾讯云 特定地域
	ProjectID string `json:"project_id,omitempty"` // GCP / 华为云 项目 ID
	UserEmail string `json:"user_email,omitempty"` // 账户主邮箱 (如 Cloudflare Global API Key 或 GCP)
}

// Settings 全局配置
type Settings struct {
	IsPinConfigured bool   `json:"is_pin_configured"`
	PinLength       int    `json:"pin_length,omitempty"`   // 记住 PIN 位数以支持盲打自适应
	PinSalt         string `json:"pin_salt,omitempty"`     // Base64 格式
	PinVerifier     string `json:"pin_verifier,omitempty"` // Base64 格式的加密金丝雀

	AutoLockMinutes int    `json:"auto_lock_minutes"` // 自动锁定时间（分钟），0 表示不自动锁
	ProxyEnabled    bool   `json:"proxy_enabled"`
	ProxyURL        string `json:"proxy_url"`                 // 如 socks5://127.0.0.1:7890 或 http://127.0.0.1:7890
	ProxyRouting    string `json:"proxy_routing"`             // smart / all
	Theme           string `json:"theme"`                     // auto / light / dark (暗黑模式)

	// 全选聚合与域名分页展示
	AggregateAllDefault   bool     `json:"aggregate_all_default"`   // 默认进入时全选所有云厂商 (默认 true)
	ZonePageSize          int      `json:"zone_page_size"`          // 域名列表每页数量 (10, 20, 50, 100, 200，默认 20)
	AutoHideMismatchedNS  bool     `json:"auto_hide_mismatched_ns"` // 自动隐藏权威 NS 不属于当前厂商的无效域名
	HiddenZones           []string `json:"hidden_zones"`            // 手动隐藏的域名列表，格式: account_id:zone_name
}

// DefaultSettings 默认配置
func DefaultSettings() Settings {
	return Settings{
		IsPinConfigured:      false,
		AutoLockMinutes:      15,
		ProxyEnabled:         false,
		ProxyURL:             "",
		ProxyRouting:         ProxyRoutingSmart,
		Theme:                ThemeAuto,
		AggregateAllDefault:  true,
		ZonePageSize:         20,
		AutoHideMismatchedNS: false,
		HiddenZones:          []string{},
	}
}
