package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TencentCloudProvider 腾讯云官方 API 3.0 提供者 (基于 TC3-HMAC-SHA256 签名)
type TencentCloudProvider struct {
	client        *http.Client
	secretId      string
	secretKey     string
	accountID     string
	providerName  string
	enableEdgeOne bool
}

func NewTencentCloudProvider(client *http.Client, secretId, secretKey, accountID string, enableEdgeOne bool) *TencentCloudProvider {
	return NewTencentCloudProviderWithCustomName(client, secretId, secretKey, accountID, "tencent_cloud", enableEdgeOne)
}

func NewTencentCloudProviderWithCustomName(client *http.Client, secretId, secretKey, accountID, providerName string, enableEdgeOne bool) *TencentCloudProvider {
	if providerName == "" {
		providerName = "tencent_cloud"
	}
	return &TencentCloudProvider{
		client:        client,
		secretId:      strings.TrimSpace(secretId),
		secretKey:     strings.TrimSpace(secretKey),
		accountID:     accountID,
		providerName:  providerName,
		enableEdgeOne: enableEdgeOne,
	}
}

func hmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

// doRequest 发送经过腾讯云 TC3-HMAC-SHA256 标准签名的请求
func (p *TencentCloudProvider) doRequest(ctx context.Context, service, version, action string, payload map[string]interface{}) ([]byte, error) {
	host := service + ".tencentcloudapi.com"
	endpoint := "https://" + host

	if payload == nil {
		payload = make(map[string]interface{})
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求参数失败: %w", err)
	}
	bodyStr := string(bodyBytes)

	now := time.Now().UTC()
	timestamp := now.Unix()
	date := now.Format("2006-01-02")

	// 1. 规范请求串 (CanonicalRequest)
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:application/json; charset=utf-8\nhost:%s\nx-tc-action:%s\n", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	hashedRequestPayload := sha256Hex(bodyStr)

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload,
	)

	// 2. 待签名字符串 (StringToSign)
	algorithm := "TC3-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256Hex(canonicalRequest)

	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest,
	)

	// 3. 计算派生签名密钥与签名 (Signature)
	secretDate := hmacSha256([]byte("TC3"+p.secretKey), date)
	secretService := hmacSha256(secretDate, service)
	secretSigning := hmacSha256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSha256(secretSigning, stringToSign))

	// 4. 构建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Authorization", fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, p.secretId, credentialScope, signedHeaders, signature))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyTrimmed := strings.TrimSpace(string(body))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("腾讯云 API 响应状态异常 [%d]: %s", resp.StatusCode, bodyTrimmed)
	}

	if strings.HasPrefix(bodyTrimmed, "<") {
		return nil, fmt.Errorf("腾讯云返回非 JSON 响应 (可能是网络拦截或鉴权被拒): %s", bodyTrimmed)
	}

	// 5. 校验业务层错误
	var errCheck struct {
		Response struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RequestId string `json:"RequestId"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &errCheck); err == nil && errCheck.Response.Error.Code != "" {
		reqIdPart := ""
		if errCheck.Response.RequestId != "" {
			reqIdPart = fmt.Sprintf(" (RequestId: %s)", errCheck.Response.RequestId)
		}
		return nil, fmt.Errorf("腾讯云 [%s.%s] 鉴权或操作失败 [%s]: %s%s", service, action, errCheck.Response.Error.Code, errCheck.Response.Error.Message, reqIdPart)
	}

	return body, nil
}

// TestConnection 探测凭据可用性
func (p *TencentCloudProvider) TestConnection(ctx context.Context) error {
	// 腾讯云 EdgeOne 独立服务商：仅探测 EdgeOne
	if p.providerName == "tencent_edgeone" {
		_, errTEO := p.doRequest(ctx, "teo", "2022-09-01", "DescribeZones", map[string]interface{}{
			"Limit": 1,
		})
		return errTEO
	}

	// 腾讯云云解析 DNS / DNSPod 模式：探测云解析
	_, err := p.doRequest(ctx, "dnspod", "2021-03-23", "DescribeDomainList", map[string]interface{}{
		"Limit": 1,
	})
	return err
}

type tcDomainListResp struct {
	Response struct {
		DomainList []struct {
			DomainId any    `json:"DomainId"`
			Name     string `json:"Name"`
			Status   string `json:"Status"`
		} `json:"DomainList"`
		TotalCount int `json:"TotalCount"`
	} `json:"Response"`
}

type tcTEOZoneListResp struct {
	Response struct {
		Zones []struct {
			ZoneId      string   `json:"ZoneId"`
			ZoneName    string   `json:"ZoneName"`
			Type        string   `json:"Type"`       // full: NS 接入, partial: CNAME 接入, dnsPodAccess: DNSPod托管
			ZoneType    string   `json:"ZoneType"`   // 备用兼容字段
			AccessType  string   `json:"AccessType"` // 备用接入方式字段
			Status      string   `json:"Status"`
			NameServers []string `json:"NameServers"`
		} `json:"Zones"`
		TotalCount int `json:"TotalCount"`
	} `json:"Response"`
}

// ListZones 获取域名列表
func (p *TencentCloudProvider) ListZones(ctx context.Context) ([]Zone, error) {
	var zones []Zone

	// 1. 若为腾讯云 EdgeOne 独立服务商，仅拉取 EdgeOne 站点 (包含 NS 接入与 CNAME 接入)
	if p.providerName == "tencent_edgeone" {
		teoOffset := 0
		teoLimit := 100
		for {
			teoBody, tErr := p.doRequest(ctx, "teo", "2022-09-01", "DescribeZones", map[string]interface{}{
				"Offset": teoOffset,
				"Limit":  teoLimit,
			})
			if tErr != nil {
				return nil, tErr
			}

			var teoParsed tcTEOZoneListResp
			if jsonErr := json.Unmarshal(teoBody, &teoParsed); jsonErr != nil {
				return nil, fmt.Errorf("解析 EdgeOne 列表失败: %w", jsonErr)
			}

			if len(teoParsed.Response.Zones) == 0 {
				break
			}

			for _, z := range teoParsed.Response.Zones {
				zoneType := strings.ToLower(z.Type)
				if zoneType == "" {
					zoneType = strings.ToLower(z.ZoneType)
				}
				if zoneType == "" {
					zoneType = strings.ToLower(z.AccessType)
				}
				zNameLower := strings.ToLower(z.ZoneName)

				// 仅排除无域名内部虚拟站点 (如 Pages 虚拟站点)
				isPages := zoneType == "partialcomposite" || strings.Contains(zNameLower, "pages-zone")
				if isPages {
					continue
				}

				// 接入类型识别：CNAME 接入 vs NS 接入
				accessType := "NS"
				if zoneType == "partial" || strings.Contains(zoneType, "cname") {
					accessType = "CNAME"
				}

				// 状态判定：CNAME 接入模式无需修改权威 NS，直接认定为 matched 正常；NS 接入模式若官方 active 则认定为 matched
				nsStatus := ""
				if accessType == "CNAME" || strings.ToLower(z.Status) == "active" {
					nsStatus = "matched"
				}

				zones = append(zones, Zone{
					ID:               z.ZoneId,
					Name:             z.ZoneName,
					Status:           z.Status,
					Provider:         "tencent_edgeone",
					AccountID:        p.accountID,
					AccessType:       accessType,
					ActualNS:         z.NameServers,
					NSStatus:         nsStatus,
					DetectedProvider: "tencent_edgeone",
				})
			}

			teoOffset += len(teoParsed.Response.Zones)
			if teoOffset >= teoParsed.Response.TotalCount || len(teoParsed.Response.Zones) < teoLimit {
				break
			}
		}
		return zones, nil
	}

	// 2. 腾讯云云解析 DNS / DNSPod 域名列表 (官方规定 Limit 最大 100，使用分页拉取)
	offset := 0
	limit := 100
	for {
		body, err := p.doRequest(ctx, "dnspod", "2021-03-23", "DescribeDomainList", map[string]interface{}{
			"Type":   "ALL",
			"Offset": offset,
			"Limit":  limit,
		})
		if err != nil {
			return nil, err
		}

		var parsed tcDomainListResp
		if jsonErr := json.Unmarshal(body, &parsed); jsonErr != nil {
			return nil, fmt.Errorf("解析云解析 DNS 列表失败: %w", jsonErr)
		}

		if len(parsed.Response.DomainList) == 0 {
			break
		}

		for _, d := range parsed.Response.DomainList {
			status := "active"
			if d.Status != "ENABLE" {
				status = d.Status
			}
			zones = append(zones, Zone{
				ID:         fmt.Sprintf("%v", d.DomainId),
				Name:       d.Name,
				Status:     status,
				Provider:   p.providerName,
				AccountID:  p.accountID,
				AccessType: "NS",
			})
		}

		offset += len(parsed.Response.DomainList)
		if offset >= parsed.Response.TotalCount || len(parsed.Response.DomainList) < limit {
			break
		}
	}

	return zones, nil
}

type tcRecordListResp struct {
	Response struct {
		RecordList []struct {
			RecordId any    `json:"RecordId"`
			Name     string `json:"Name"`
			Type     string `json:"Type"`
			Value    string `json:"Value"`
			TTL      any    `json:"TTL"`
			MX       any    `json:"MX"`
			Remark   string `json:"Remark"`
			Status   string `json:"Status"`
		} `json:"RecordList"`
	} `json:"Response"`
}

type tcTEORecordListResp struct {
	Response struct {
		DnsRecords []struct {
			RecordId string `json:"RecordId"`
			Name     string `json:"Name"`
			Type     string `json:"Type"`
			Content  string `json:"Content"`
			TTL      any    `json:"TTL"`
			Priority any    `json:"Priority"`
		} `json:"DnsRecords"`
	} `json:"Response"`
}

type tcTEOAccelerationDomainResp struct {
	Response struct {
		TotalCount          int `json:"TotalCount"`
		AccelerationDomains []struct {
			ZoneId       string `json:"ZoneId"`
			DomainName   string `json:"DomainName"`
			DomainStatus string `json:"DomainStatus"`
			Cname        string `json:"Cname"`
		} `json:"AccelerationDomains"`
	} `json:"Response"`
}

// ListRecords 获取解析记录
func (p *TencentCloudProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	// 如果是 EdgeOne 站点 (ZoneID 以 zone- 开头)
	if strings.HasPrefix(zoneID, "zone-") {
		records := make([]Record, 0)

		// 1. 尝试获取 EdgeOne DNS 解析记录 (针对 NS 托管站点)
		body, err := p.doRequest(ctx, "teo", "2022-09-01", "DescribeDnsRecords", map[string]interface{}{
			"ZoneId": zoneID,
			"Limit":  1000,
		})
		if err == nil {
			var parsed tcTEORecordListResp
			if jsonErr := json.Unmarshal(body, &parsed); jsonErr == nil {
				for _, r := range parsed.Response.DnsRecords {
					ttlInt, _ := strconv.Atoi(fmt.Sprintf("%v", r.TTL))
					priInt, _ := strconv.Atoi(fmt.Sprintf("%v", r.Priority))
					records = append(records, Record{
						ID:       r.RecordId,
						ZoneID:   zoneID,
						ZoneName: zoneName,
						Type:     r.Type,
						Name:     r.Name,
						Content:  r.Content,
						TTL:      ttlInt,
						Priority: priInt,
					})
				}
			}
		}

		// 2. 额外尝试获取加速域名 (针对 CNAME 接入的站点)
		accBody, accErr := p.doRequest(ctx, "teo", "2022-09-01", "DescribeAccelerationDomains", map[string]interface{}{
			"ZoneId": zoneID,
			"Limit":  200,
		})
		if accErr == nil {
			var accParsed tcTEOAccelerationDomainResp
			if jsonErr := json.Unmarshal(accBody, &accParsed); jsonErr == nil {
				for _, d := range accParsed.Response.AccelerationDomains {
					if d.DomainName == "" && d.Cname == "" {
						continue
					}
					statusDesc := d.DomainStatus
					switch d.DomainStatus {
					case "online":
						statusDesc = "已生效"
					case "process":
						statusDesc = "部署中"
					case "offline":
						statusDesc = "已停用"
					}
					records = append(records, Record{
						ID:       "acc-" + d.DomainName,
						ZoneID:   zoneID,
						ZoneName: zoneName,
						Type:     "CNAME",
						Name:     d.DomainName,
						Content:  d.Cname,
						TTL:      600,
						Comment:  fmt.Sprintf("EdgeOne CNAME 加速 (%s)", statusDesc),
					})
				}
			}
		}

		return records, nil
	}

	// 腾讯云云解析 DNS (API 3.0)
	payload := map[string]interface{}{
		"Domain": zoneName,
		"Limit":  3000,
	}
	if did, err := strconv.ParseUint(zoneID, 10, 64); err == nil && did > 0 {
		payload["DomainId"] = did
	}
	body, err := p.doRequest(ctx, "dnspod", "2021-03-23", "DescribeRecordList", payload)
	if err != nil {
		return nil, err
	}

	var parsed tcRecordListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析记录列表失败: %w", err)
	}

	records := make([]Record, len(parsed.Response.RecordList))
	for i, r := range parsed.Response.RecordList {
		ttlInt, _ := strconv.Atoi(fmt.Sprintf("%v", r.TTL))
		mxInt, _ := strconv.Atoi(fmt.Sprintf("%v", r.MX))

		records[i] = Record{
			ID:       fmt.Sprintf("%v", r.RecordId),
			ZoneID:   zoneID,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     r.Name,
			Content:  r.Value,
			TTL:      ttlInt,
			Priority: mxInt,
			Comment:  r.Remark,
		}
	}

	return records, nil
}

// CreateRecord 创建解析记录
func (p *TencentCloudProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	if strings.HasPrefix(zoneID, "zone-") {
		payload := map[string]interface{}{
			"ZoneId":  zoneID,
			"Name":    r.Name,
			"Type":    r.Type,
			"Content": r.Content,
		}
		if r.TTL > 0 {
			payload["TTL"] = r.TTL
		}
		if r.Priority > 0 {
			payload["Priority"] = r.Priority
		}
		body, err := p.doRequest(ctx, "teo", "2022-09-01", "CreateDnsRecord", payload)
		if err != nil {
			return nil, err
		}
		var resp struct {
			Response struct {
				RecordId string `json:"RecordId"`
			} `json:"Response"`
		}
		_ = json.Unmarshal(body, &resp)
		r.ID = resp.Response.RecordId
		r.ZoneID = zoneID
		r.ZoneName = zoneName
		return &r, nil
	}

	payload := map[string]interface{}{
		"Domain":     zoneName,
		"SubDomain":  r.Name,
		"RecordType": r.Type,
		"RecordLine": "默认",
		"Value":      r.Content,
	}
	if r.TTL > 0 {
		payload["TTL"] = r.TTL
	}
	if r.Priority > 0 {
		payload["MX"] = r.Priority
	}
	if r.Comment != "" {
		payload["Remark"] = r.Comment
	}

	body, err := p.doRequest(ctx, "dnspod", "2021-03-23", "CreateRecord", payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response struct {
			RecordId any `json:"RecordId"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析创建记录响应失败: %w", err)
	}

	r.ID = fmt.Sprintf("%v", resp.Response.RecordId)
	r.ZoneID = zoneID
	r.ZoneName = zoneName
	return &r, nil
}

// UpdateRecord 修改解析记录
func (p *TencentCloudProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	if strings.HasPrefix(zoneID, "zone-") {
		payload := map[string]interface{}{
			"ZoneId":   zoneID,
			"RecordId": r.ID,
			"Name":     r.Name,
			"Type":     r.Type,
			"Content":  r.Content,
		}
		if r.TTL > 0 {
			payload["TTL"] = r.TTL
		}
		if r.Priority > 0 {
			payload["Priority"] = r.Priority
		}
		_, err := p.doRequest(ctx, "teo", "2022-09-01", "ModifyDnsRecord", payload)
		if err != nil {
			return nil, err
		}
		return &r, nil
	}

	recIDInt, _ := strconv.ParseUint(r.ID, 10, 64)
	payload := map[string]interface{}{
		"Domain":     zoneName,
		"RecordId":   recIDInt,
		"SubDomain":  r.Name,
		"RecordType": r.Type,
		"RecordLine": "默认",
		"Value":      r.Content,
	}
	if r.TTL > 0 {
		payload["TTL"] = r.TTL
	}
	if r.Priority > 0 {
		payload["MX"] = r.Priority
	}
	if r.Comment != "" {
		payload["Remark"] = r.Comment
	}

	_, err := p.doRequest(ctx, "dnspod", "2021-03-23", "ModifyRecord", payload)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// DeleteRecord 删除解析记录
func (p *TencentCloudProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	if strings.HasPrefix(zoneID, "zone-") {
		_, err := p.doRequest(ctx, "teo", "2022-09-01", "DeleteDnsRecords", map[string]interface{}{
			"ZoneId":    zoneID,
			"RecordIds": []string{recordID},
		})
		return err
	}

	recIDInt, _ := strconv.ParseUint(recordID, 10, 64)
	_, err := p.doRequest(ctx, "dnspod", "2021-03-23", "DeleteRecord", map[string]interface{}{
		"Domain":   zoneName,
		"RecordId": recIDInt,
	})
	return err
}
