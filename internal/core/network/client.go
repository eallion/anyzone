package network

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// ClientOptions 客户端构建参数
type ClientOptions struct {
	ProxyEnabled bool
	ProxyURL     string
	ProxyRouting string // smart 或 all
	IsDomestic   bool   // 是否为国内厂商
}

// BuildHttpClient 根据代理设置与厂商归属构建 http.Client
func BuildHttpClient(opts ClientOptions) *http.Client {
	timeout := 15 * time.Second

	// 若未开启代理，或在 smart 模式下是国内厂商，则使用直连
	shouldUseProxy := opts.ProxyEnabled && opts.ProxyURL != ""
	if opts.ProxyRouting == "smart" && opts.IsDomestic {
		shouldUseProxy = false
	}

	if !shouldUseProxy {
		return &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}
	}

	parsedURL, err := url.Parse(opts.ProxyURL)
	if err != nil {
		// 代理地址解析失败，降级回直连
		return &http.Client{Timeout: timeout}
	}

	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme == "socks5" || scheme == "socks5h" {
		var auth *proxy.Auth
		if parsedURL.User != nil {
			auth = &proxy.Auth{
				User: parsedURL.User.Username(),
			}
			if password, ok := parsedURL.User.Password(); ok {
				auth.Password = password
			}
		}

		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, proxy.Direct)
		if err == nil {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
	} else {
		// HTTP / HTTPS 代理
		transport.Proxy = http.ProxyURL(parsedURL)
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
