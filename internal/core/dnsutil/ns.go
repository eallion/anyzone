package dnsutil

import (
	"context"
	"net"
	"strings"
	"time"
)

// NSCheckResult 域名权威 NS 检测结果
type NSCheckResult struct {
	AccountID        string   `json:"account_id,omitempty"`       // 归属账号 ID，实现多厂商多账号同名域名严格隔离
	ExpectedProvider string   `json:"expected_provider,omitempty"`// 预期的云厂商类型
	Domain           string   `json:"domain"`
	ActualNS         []string `json:"actual_ns"`          // 公网查询到的权威 Name Servers
	IsMatched        bool     `json:"is_matched"`         // 是否与当前云厂商匹配
	DetectedProvider string   `json:"detected_provider"`  // 识别出的实际托管商名称
	Status           string   `json:"status"`             // matched, mismatched, query_failed, unknown
	ErrorMessage     string   `json:"error_message,omitempty"`
}

// 常见云厂商与权威 NS 特征库
var providerNSSuffixes = map[string][]string{
	"cloudflare":       {"cloudflare.com"},
	"aliyun":           {"alidns.com", "hichina.com", "alibaba.com", "alibabadns.com", "alicloud.com", "alicloud-dns.com"},
	"aliyun_esa":       {"alidns.com", "hichina.com", "alibaba.com", "alibabadns.com", "alicloud.com", "alicloud-dns.com"},
	"dnspod":           {"dnspod.net", "dnspod.com", "dnspod.cn", "tencentcloud.com", "tencentcloud.net", "tencent.com", "qcloud.com"},
	"tencent_cloud":    {"dnspod.net", "dnspod.com", "dnspod.cn", "tencentcloud.com", "tencentcloud.net", "tencent.com", "qcloud.com"},
	"tencent_edgeone":  {"teodns.com", "teodns.net", "teodns.cn", "edgeone.ai", "edgeone.com", "edgeone.cn", "dnspod.net", "dnspod.com", "dnspod.cn", "tencentcloud.com", "tencentcloud.net", "tencent.com", "qcloud.com"},
	"huawei":           {"hwclouds-dns.com", "huaweicloud-dns.com", "hwclouds-dns.cn", "huaweicloud-dns.net"},
	"volcengine":       {"volces.com", "volcengine.com"},
	"baidu":            {"bdydns.com", "baidubce.com"},
	"aws_route53":      {"awsdns"},
	"gcp_dns":          {"googledomains.com"},
	"azure_dns":        {"azure-dns.com", "azure-dns.net"},
	"oci_dns":          {"oraclecloud.com"},
	"porkbun":          {"porkbun.com"},
	"hetzner":          {"hetzner.com", "hetzner.de"},
	"digitalocean":     {"digitalocean.com"},
	"godaddy":          {"domaincontrol.com"},
	"namecheap":        {"registrar-servers.com"},
	"dnsimple":         {"dnsimple.com"},
	"bunny":            {"b-cdn.net", "bunny.net"},
	"desec":            {"desec.io"},
	"powerdns":         {}, // 自建私有，不限特定公网后缀
	"bind9_rfc2136":    {},
	"adguard_home":     {},
}

// CheckDomainNS 查询并核验域名的权威 Name Server
func CheckDomainNS(ctx context.Context, domain string, expectedProvider string) NSCheckResult {
	res := NSCheckResult{
		Domain:           domain,
		ExpectedProvider: expectedProvider,
		Status:           "unknown",
	}

	cleanDomain := strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if cleanDomain == "" {
		res.Status = "query_failed"
		res.ErrorMessage = "域名为空"
		return res
	}

	// 自建私有 DNS，无需也无法通过公网 NS 规则判断
	if expectedProvider == "powerdns" || expectedProvider == "bind9_rfc2136" || expectedProvider == "adguard_home" {
		res.IsMatched = true
		res.Status = "matched"
		res.DetectedProvider = expectedProvider
		return res
	}

	// 创建带超时的 DNS 解析器，使用公共权威解析
	resolver := &net.Resolver{
		PreferGo: true,
	}

	queryCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
	defer cancel()

	nss, err := resolver.LookupNS(queryCtx, cleanDomain)
	if err != nil {
		res.Status = "query_failed"
		res.ErrorMessage = err.Error()
		// 查询失败时不盲目判死刑，保留原有可用性
		res.IsMatched = true
		return res
	}

	var nsList []string
	for _, ns := range nss {
		host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(ns.Host), "."))
		if host != "" {
			nsList = append(nsList, host)
		}
	}
	res.ActualNS = nsList

	if len(nsList) == 0 {
		res.Status = "query_failed"
		res.ErrorMessage = "未查到公网权威 NS 记录"
		res.IsMatched = false
		return res
	}

	// 1. 判断是否匹配预期厂商
	expectedPatterns, hasProvider := providerNSSuffixes[expectedProvider]
	matched := false
	if hasProvider && len(expectedPatterns) > 0 {
		for _, ns := range nsList {
			for _, pat := range expectedPatterns {
				if strings.Contains(ns, pat) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
	} else {
		// 没有预设规则的未知厂商，默认信任
		matched = true
	}

	res.IsMatched = matched

	// 2. 尝试从公网 NS 中识别出实际是哪家厂商
	detected := "other"
	for pKey, pats := range providerNSSuffixes {
		for _, ns := range nsList {
			for _, pat := range pats {
				if strings.Contains(ns, pat) {
					detected = pKey
					break
				}
			}
			if detected != "other" {
				break
			}
		}
		if detected != "other" {
			break
		}
	}
	if matched {
		res.Status = "matched"
		if expectedProvider != "" {
			res.DetectedProvider = expectedProvider
		}
	} else {
		res.Status = "mismatched"
	}

	return res
}
