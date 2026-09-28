package provider

import (
	"context"
)

// Record 代表通用统一的 DNS 解析记录
type Record struct {
	ID        string `json:"id"`
	ZoneID    string `json:"zone_id"`
	ZoneName  string `json:"zone_name"`
	Type      string `json:"type"`      // A, AAAA, CNAME, TXT, MX, NS, SRV, CAA 等
	Name      string `json:"name"`      // 主机记录，如 @ 或 www 或子域名
	Content   string `json:"content"`   // 记录值（IP、目标域名、文本等）
	TTL       int    `json:"ttl"`       // 秒，1 通常代表 Cloudflare 自动
	Priority  int    `json:"priority,omitempty"` // MX 或 SRV 优先级
	Proxied   *bool  `json:"proxied,omitempty"`  // Cloudflare 专属 CDN 加速开关
	Comment   string `json:"comment,omitempty"`  // 备注说明
}

// Zone 代表一个主域名空间
type Zone struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	Provider         string   `json:"provider"`
	AccountID        string   `json:"account_id"`
	AccountName      string   `json:"account_name,omitempty"`      // 聚合视图中显示的所属账号名称
	ActualNS         []string `json:"actual_ns,omitempty"`         // 探测到的权威 Name Servers
	NSStatus         string   `json:"ns_status,omitempty"`         // matched (正常生效), mismatched (未委派/已转出), query_failed, unknown
	DetectedProvider string   `json:"detected_provider,omitempty"` // 实际 NS 归属厂商
	IsHidden         bool     `json:"is_hidden,omitempty"`         // 是否已被用户或规则隐藏
	AccessType       string   `json:"access_type,omitempty"`       // 接入方式：NS 接入或 CNAME 接入
}

// DNSProvider 通用接口
type DNSProvider interface {
	// TestConnection 测试凭据与连通性
	TestConnection(ctx context.Context) error

	// ListZones 获取当前账号下的所有域名
	ListZones(ctx context.Context) ([]Zone, error)

	// ListRecords 获取指定域名下的全部解析记录
	ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error)

	// CreateRecord 创建解析记录
	CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error)

	// UpdateRecord 更新解析记录
	UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error)

	// DeleteRecord 删除指定解析记录
	DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error
}
