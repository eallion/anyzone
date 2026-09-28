package provider

import (
	"fmt"
	"net/http"

	"anyzone/internal/core/config"
	"anyzone/internal/core/network"
)

// CreateProvider 根据账号、解密凭据及代理配置生成对应的 DNSProvider 实例
func CreateProvider(account config.Account, creds config.Credentials, settings config.Settings) (DNSProvider, error) {
	isDomestic := account.Provider == config.ProviderAliyun ||
		account.Provider == config.ProviderAliyunESA ||
		account.Provider == config.ProviderDNSPod ||
		account.Provider == config.ProviderTencentCloud ||
		account.Provider == config.ProviderTencentEdgeOne ||
		account.Provider == config.ProviderHuawei ||
		account.Provider == config.ProviderVolcengine ||
		account.Provider == config.ProviderBaidu ||
		account.Provider == config.ProviderPowerDNS ||
		account.Provider == config.ProviderBIND9 ||
		account.Provider == config.ProviderAdGuard

	proxyURL := settings.ProxyURL
	if account.CustomProxy != "" {
		proxyURL = account.CustomProxy
	}

	httpClient := network.BuildHttpClient(network.ClientOptions{
		ProxyEnabled: settings.ProxyEnabled,
		ProxyURL:     proxyURL,
		ProxyRouting: settings.ProxyRouting,
		IsDomestic:   isDomestic,
	})

	return CreateProviderWithClient(account, creds, httpClient)
}

func CreateProviderWithClient(account config.Account, creds config.Credentials, client *http.Client) (DNSProvider, error) {
	switch account.Provider {
	// 1. Cloudflare
	case config.ProviderCloudflare:
		if creds.ApiToken == "" {
			return nil, fmt.Errorf("Cloudflare 缺少 API Token")
		}
		return NewCloudflareProvider(client, creds.ApiToken, account.ID), nil

	// 2. 阿里云 (Alidns 纯云解析 DNS)
	case config.ProviderAliyun:
		ak := creds.AccessKeyId
		if ak == "" {
			ak = creds.ApiKey
		}
		sk := creds.AccessKeySecret
		if sk == "" {
			sk = creds.ApiSecret
		}
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("阿里云 (Alidns) 缺少 AccessKey ID 或 AccessKey Secret")
		}
		return NewAliyunProvider(client, ak, sk, account.ID, config.ProviderAliyun), nil

	// 2.1 阿里云 ESA (边缘安全加速平台)
	case config.ProviderAliyunESA:
		ak := creds.AccessKeyId
		if ak == "" {
			ak = creds.ApiKey
		}
		sk := creds.AccessKeySecret
		if sk == "" {
			sk = creds.ApiSecret
		}
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("阿里云 ESA 缺少 AccessKey ID 或 AccessKey Secret")
		}
		p := NewAliyunProvider(client, ak, sk, account.ID, config.ProviderAliyunESA)
		if creds.Region != "" {
			p.SetRegion(creds.Region)
		}
		return p, nil

	// 3. 腾讯云 (纯云解析 DNS)
	case config.ProviderTencentCloud:
		secretId := creds.AccessKeyId
		if secretId == "" {
			secretId = creds.ApiKey
		}
		secretKey := creds.AccessKeySecret
		if secretKey == "" {
			secretKey = creds.ApiSecret
		}
		if secretId != "" && secretKey != "" {
			return NewTencentCloudProviderWithCustomName(client, secretId, secretKey, account.ID, config.ProviderTencentCloud, false), nil
		}
		if creds.TokenId != "" && creds.Token != "" {
			return NewDNSPodProvider(client, creds.TokenId, creds.Token, account.ID, false), nil
		}
		return nil, fmt.Errorf("腾讯云缺少 SecretId 或 SecretKey (CAM 访问密钥)")

	// 3.1 腾讯云 EdgeOne (边缘安全加速平台)
	case config.ProviderTencentEdgeOne:
		secretId := creds.AccessKeyId
		if secretId == "" {
			secretId = creds.ApiKey
		}
		secretKey := creds.AccessKeySecret
		if secretKey == "" {
			secretKey = creds.ApiSecret
		}
		if secretId == "" || secretKey == "" {
			return nil, fmt.Errorf("腾讯云 EdgeOne 缺少 SecretId 或 SecretKey (CAM 访问密钥)")
		}
		return NewTencentCloudProviderWithCustomName(client, secretId, secretKey, account.ID, config.ProviderTencentEdgeOne, true), nil

	// 3.2 DNSPod (独立支持 DNSPod Token 与 腾讯云 API 密钥两种认证方式)
	case config.ProviderDNSPod:
		if creds.TokenId != "" && creds.Token != "" {
			return NewDNSPodProvider(client, creds.TokenId, creds.Token, account.ID, false), nil
		}
		secretId := creds.AccessKeyId
		if secretId == "" {
			secretId = creds.ApiKey
		}
		secretKey := creds.AccessKeySecret
		if secretKey == "" {
			secretKey = creds.ApiSecret
		}
		if secretId != "" && secretKey != "" {
			return NewTencentCloudProviderWithCustomName(client, secretId, secretKey, account.ID, config.ProviderDNSPod, false), nil
		}
		return nil, fmt.Errorf("DNSPod 缺少有效认证信息：请填写 DNSPod Token (ID+Token) 或 腾讯云 API 密钥 (SecretId+SecretKey)")

	// 4. 华为云 DNS
	case config.ProviderHuawei:
		token := creds.ApiToken
		if token == "" {
			token = creds.Token
		}
		if token == "" {
			return nil, fmt.Errorf("华为云缺少 IAM 用户 API Token (X-Auth-Token)")
		}
		return NewHuaweiProvider(client, token, account.ID), nil

	// 5. 火山引擎 (字节跳动) / 百度智能云
	case config.ProviderVolcengine, config.ProviderBaidu:
		ak := creds.AccessKeyId
		if ak == "" {
			ak = creds.ApiKey
		}
		sk := creds.AccessKeySecret
		if sk == "" {
			sk = creds.ApiSecret
		}
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("缺少 AccessKey ID 或 Secret")
		}
		return NewAliyunProvider(client, ak, sk, account.ID, config.ProviderAliyun), nil

	// 6. Porkbun
	case config.ProviderPorkbun:
		key := creds.ApiKey
		if key == "" {
			key = creds.AccessKeyId
		}
		sec := creds.SecretApiKey
		if sec == "" {
			sec = creds.ApiSecret
		}
		if key == "" || sec == "" {
			return nil, fmt.Errorf("Porkbun 缺少 API Key 或 Secret API Key")
		}
		return NewPorkbunProvider(client, key, sec, account.ID), nil

	// 7. Hetzner DNS
	case config.ProviderHetzner:
		if creds.ApiToken == "" {
			return nil, fmt.Errorf("Hetzner 缺少 Auth-API-Token")
		}
		return NewHetznerProvider(client, creds.ApiToken, account.ID), nil

	// 8. DigitalOcean / Vultr / Linode / Bunny.net / deSEC
	case config.ProviderDigitalOcean, config.ProviderVultr, config.ProviderLinode, config.ProviderBunny, config.ProviderDeSEC, config.ProviderDNSimple:
		token := creds.ApiToken
		if token == "" {
			token = creds.ApiKey
		}
		if token == "" {
			return nil, fmt.Errorf("缺少 API Bearer / Personal Access Token")
		}
		return NewDigitalOceanProvider(client, token, account.ID), nil

	// 9. GoDaddy / AWS Route 53 / Namecheap
	case config.ProviderGoDaddy, config.ProviderAWSRoute53, config.ProviderNamecheap, config.ProviderGoogleCloud, config.ProviderAzureDNS, config.ProviderOracleCloud:
		key := creds.ApiKey
		if key == "" {
			key = creds.AccessKeyId
		}
		secret := creds.ApiSecret
		if secret == "" {
			secret = creds.AccessKeySecret
		}
		if key == "" || secret == "" {
			return nil, fmt.Errorf("缺少 API Key / AccessKeyId 或 Secret")
		}
		return NewGoDaddyProvider(client, key, secret, account.ID), nil

	// 10. 自建与私有服务 (PowerDNS / BIND 9 / AdGuard)
	case config.ProviderPowerDNS, config.ProviderBIND9, config.ProviderAdGuard:
		url := creds.ServerUrl
		token := creds.ApiToken
		if token == "" {
			token = creds.ApiKey
		}
		if url == "" || token == "" {
			return nil, fmt.Errorf("自建/私有 DNS 缺少服务器地址 (ServerUrl) 或 API Key")
		}
		return NewPowerDNSProvider(client, url, token, account.ID), nil

	default:
		return nil, fmt.Errorf("暂不支持的 DNS 厂商类型: %s", account.Provider)
	}
}
