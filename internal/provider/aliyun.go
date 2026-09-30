package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	aliyunEndpoint           = "https://alidns.aliyuncs.com"
	aliyunESADefaultEndpoint = "https://esa.cn-hangzhou.aliyuncs.com"
)

type AliyunProvider struct {
	client          *http.Client
	accessKeyId     string
	accessKeySecret string
	accountID       string
	providerType    string // "aliyun" 或 "aliyun_esa"
	region          string
}

func NewAliyunProvider(client *http.Client, ak, sk, accountID string, providerType string) *AliyunProvider {
	if providerType == "" {
		providerType = "aliyun"
	}
	return &AliyunProvider{
		client:          client,
		accessKeyId:     ak,
		accessKeySecret: sk,
		accountID:       accountID,
		providerType:    providerType,
	}
}

func (p *AliyunProvider) SetRegion(region string) {
	p.region = strings.TrimSpace(region)
}

func (p *AliyunProvider) getESAEndpoint() string {
	if p.region != "" {
		return fmt.Sprintf("https://esa.%s.aliyuncs.com", p.region)
	}
	return aliyunESADefaultEndpoint
}

func specialUrlEncode(s string) string {
	res := url.QueryEscape(s)
	res = strings.ReplaceAll(res, "+", "%20")
	res = strings.ReplaceAll(res, "*", "%2A")
	res = strings.ReplaceAll(res, "%7E", "~")
	return res
}

func (p *AliyunProvider) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalizedQuery strings.Builder
	for i, k := range keys {
		if i > 0 {
			canonicalizedQuery.WriteString("&")
		}
		canonicalizedQuery.WriteString(specialUrlEncode(k))
		canonicalizedQuery.WriteString("=")
		canonicalizedQuery.WriteString(specialUrlEncode(params[k]))
	}

	stringToSign := "GET&%2F&" + specialUrlEncode(canonicalizedQuery.String())
	mac := hmac.New(sha1.New, []byte(p.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (p *AliyunProvider) doRequest(ctx context.Context, action string, extraParams map[string]string) ([]byte, error) {
	params := map[string]string{
		"Format":           "JSON",
		"Version":          "2015-01-09",
		"AccessKeyId":      p.accessKeyId,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   strconv.FormatInt(time.Now().UnixNano(), 10),
		"Action":           action,
	}

	for k, v := range extraParams {
		params[k] = v
	}

	params["Signature"] = p.sign(params)

	uv := url.Values{}
	for k, v := range params {
		uv.Set(k, v)
	}

	reqURL := fmt.Sprintf("%s/?%s", aliyunEndpoint, uv.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			return nil, fmt.Errorf("阿里云网络请求失败: %v", urlErr.Err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var errResp struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestId string `json:"RequestId"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != "" {
		return nil, fmt.Errorf("阿里云 API 错误 [%s]: %s", errResp.Code, errResp.Message)
	}

	return body, nil
}

func (p *AliyunProvider) doESARequest(ctx context.Context, action string, extraParams map[string]string) ([]byte, error) {
	params := map[string]string{
		"Format":           "JSON",
		"Version":          "2024-09-10",
		"AccessKeyId":      p.accessKeyId,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   strconv.FormatInt(time.Now().UnixNano(), 10),
		"Action":           action,
	}

	for k, v := range extraParams {
		params[k] = v
	}

	params["Signature"] = p.sign(params)

	uv := url.Values{}
	for k, v := range params {
		uv.Set(k, v)
	}

	endpoint := p.getESAEndpoint()
	reqURL := fmt.Sprintf("%s/?%s", endpoint, uv.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			return nil, fmt.Errorf("阿里云 ESA 网络请求失败: %v", urlErr.Err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var errResp struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestId string `json:"RequestId"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != "" {
		return nil, fmt.Errorf("阿里云 ESA API 错误 [%s]: %s", errResp.Code, errResp.Message)
	}

	return body, nil
}

func (p *AliyunProvider) TestConnection(ctx context.Context) error {
	if p.providerType == "aliyun_esa" {
		_, esaErr := p.doESARequest(ctx, "ListSites", map[string]string{"PageSize": "1"})
		return esaErr
	}
	_, err := p.doRequest(ctx, "DescribeDomains", map[string]string{"PageSize": "1"})
	return err
}

type aliyunDomainListResp struct {
	Domains struct {
		Domain []struct {
			DomainId   string `json:"DomainId"`
			DomainName string `json:"DomainName"`
		} `json:"Domain"`
	} `json:"Domains"`
}

func (p *AliyunProvider) ListZones(ctx context.Context) ([]Zone, error) {
	var zones []Zone

	// 1. 若为阿里云 ESA 独立服务商，拉取 ESA 边缘安全加速站点 (包含 NS 接入与 CNAME 接入)
	if p.providerType == "aliyun_esa" {
		rawESA, errESA := p.doESARequest(ctx, "ListSites", map[string]string{
			"PageSize": "50",
		})
		if errESA != nil {
			return nil, errESA
		}
		var parsedESA struct {
			Sites []struct {
				SiteId         int64  `json:"SiteId"`
				SiteName       string `json:"SiteName"`
				Status         string `json:"Status"`
				AccessType     string `json:"AccessType"` // NS 或 CNAME
				NameServerList string `json:"NameServerList"`
			} `json:"Sites"`
		}
		if jsonErr := json.Unmarshal(rawESA, &parsedESA); jsonErr != nil {
			return nil, fmt.Errorf("解析 ESA 站点列表失败: %w", jsonErr)
		}
		for _, s := range parsedESA.Sites {
			accessType := strings.ToUpper(strings.TrimSpace(s.AccessType))
			if accessType == "" {
				accessType = "NS"
			}
			var nsList []string
			if s.NameServerList != "" {
				for _, n := range strings.Split(s.NameServerList, ",") {
					trimmed := strings.TrimSpace(n)
					if trimmed != "" {
						nsList = append(nsList, trimmed)
					}
				}
			}
			// CNAME 接入模式无需修改权威 NS，直接认定为 matched 正常；NS 接入模式若官方 active 则认定为 matched
			nsStatus := ""
			if accessType == "CNAME" || strings.ToLower(s.Status) == "active" {
				nsStatus = "matched"
			}
			zones = append(zones, Zone{
				ID:               fmt.Sprintf("esa:%d", s.SiteId),
				Name:             s.SiteName,
				Status:           s.Status,
				Provider:         "aliyun_esa",
				AccountID:        p.accountID,
				AccessType:       accessType,
				ActualNS:         nsList,
				NSStatus:         nsStatus,
				DetectedProvider: "aliyun_esa",
			})
		}
		return zones, nil
	}

	// 2. 常规阿里云云解析 DNS 域名
	raw, err := p.doRequest(ctx, "DescribeDomains", map[string]string{"PageSize": "50"})
	if err != nil {
		return nil, err
	}
	var parsed aliyunDomainListResp
	if jsonErr := json.Unmarshal(raw, &parsed); jsonErr != nil {
		return nil, fmt.Errorf("解析云解析 DNS 列表失败: %w", jsonErr)
	}
	for _, d := range parsed.Domains.Domain {
		zones = append(zones, Zone{
			ID:         d.DomainName,
			Name:       d.DomainName,
			Status:     "ENABLE",
			Provider:   "aliyun",
			AccountID:  p.accountID,
			AccessType: "NS",
		})
	}

	return zones, nil
}

type aliyunRecordListResp struct {
	DomainRecords struct {
		Record []struct {
			RecordId string `json:"RecordId"`
			RR       string `json:"RR"`
			Type     string `json:"Type"`
			Value    string `json:"Value"`
			TTL      int    `json:"TTL"`
			Priority int    `json:"Priority"`
			Remark   string `json:"Remark"`
		} `json:"Record"`
	} `json:"DomainRecords"`
}

func (p *AliyunProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	// 判断是否为 ESA 站点
	if strings.HasPrefix(zoneID, "esa:") {
		siteID := strings.TrimPrefix(zoneID, "esa:")
		raw, err := p.doESARequest(ctx, "ListRecords", map[string]string{
			"SiteId":   siteID,
			"PageSize": "100",
		})
		if err != nil {
			return nil, err
		}

		var esaResp struct {
			Records []struct {
				RecordId int64  `json:"RecordId"`
				Data     struct {
					Name  string `json:"RecordName"`
					Type  string `json:"RecordType"`
					Value string `json:"Value"`
					Ttl   int    `json:"Ttl"`
				} `json:"Data"`
				RecordName string `json:"RecordName"`
				RecordType string `json:"RecordType"`
				Value      string `json:"RecordValue"`
				Ttl        int    `json:"Ttl"`
				Comment    string `json:"Comment"`
			} `json:"Records"`
		}
		if err := json.Unmarshal(raw, &esaResp); err != nil {
			return nil, err
		}

		records := make([]Record, len(esaResp.Records))
		for i, r := range esaResp.Records {
			rName := r.RecordName
			if rName == "" {
				rName = r.Data.Name
			}
			rType := r.RecordType
			if rType == "" {
				rType = r.Data.Type
			}
			rVal := r.Value
			if rVal == "" {
				rVal = r.Data.Value
			}
			rTtl := r.Ttl
			if rTtl == 0 {
				rTtl = r.Data.Ttl
			}

			records[i] = Record{
				ID:       strconv.FormatInt(r.RecordId, 10),
				ZoneID:   zoneID,
				ZoneName: zoneName,
				Type:     rType,
				Name:     rName,
				Content:  rVal,
				TTL:      rTtl,
				Comment:  r.Comment,
			}
		}
		return records, nil
	}

	// 常规 Alidns 解析记录
	raw, err := p.doRequest(ctx, "DescribeDomainRecords", map[string]string{
		"DomainName": zoneName,
		"PageSize":   "100",
	})
	if err != nil {
		return nil, err
	}

	var parsed aliyunRecordListResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	list := parsed.DomainRecords.Record
	records := make([]Record, len(list))
	for i, r := range list {
		records[i] = Record{
			ID:       r.RecordId,
			ZoneID:   zoneName,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     r.RR,
			Content:  r.Value,
			TTL:      r.TTL,
			Priority: r.Priority,
			Comment:  r.Remark,
		}
	}
	return records, nil
}

func (p *AliyunProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	if strings.HasPrefix(zoneID, "esa:") {
		siteID := strings.TrimPrefix(zoneID, "esa:")
		params := map[string]string{
			"SiteId":      siteID,
			"RecordName":  r.Name,
			"RecordType":  r.Type,
			"RecordValue": r.Content,
			"Ttl":         strconv.Itoa(r.TTL),
		}
		if r.Comment != "" {
			params["Comment"] = r.Comment
		}
		raw, err := p.doESARequest(ctx, "CreateRecord", params)
		if err != nil {
			return nil, err
		}

		var resp struct {
			RecordId int64 `json:"RecordId"`
		}
		_ = json.Unmarshal(raw, &resp)
		return &Record{
			ID:       strconv.FormatInt(resp.RecordId, 10),
			ZoneID:   zoneID,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     r.Name,
			Content:  r.Content,
			TTL:      r.TTL,
			Comment:  r.Comment,
		}, nil
	}

	params := map[string]string{
		"DomainName": zoneName,
		"RR":         r.Name,
		"Type":       r.Type,
		"Value":      r.Content,
		"TTL":        strconv.Itoa(r.TTL),
	}
	if r.Priority > 0 {
		params["Priority"] = strconv.Itoa(r.Priority)
	}

	raw, err := p.doRequest(ctx, "AddDomainRecord", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		RecordId string `json:"RecordId"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}

	return &Record{
		ID:       resp.RecordId,
		ZoneID:   zoneName,
		ZoneName: zoneName,
		Type:     r.Type,
		Name:     r.Name,
		Content:  r.Content,
		TTL:      r.TTL,
		Priority: r.Priority,
	}, nil
}

func (p *AliyunProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	if strings.HasPrefix(zoneID, "esa:") {
		params := map[string]string{
			"RecordId":    r.ID,
			"RecordName":  r.Name,
			"RecordType":  r.Type,
			"RecordValue": r.Content,
			"Ttl":         strconv.Itoa(r.TTL),
		}
		if r.Comment != "" {
			params["Comment"] = r.Comment
		}
		_, err := p.doESARequest(ctx, "UpdateRecord", params)
		if err != nil {
			return nil, err
		}
		return &r, nil
	}

	params := map[string]string{
		"RecordId": r.ID,
		"RR":       r.Name,
		"Type":     r.Type,
		"Value":    r.Content,
		"TTL":      strconv.Itoa(r.TTL),
	}
	if r.Priority > 0 {
		params["Priority"] = strconv.Itoa(r.Priority)
	}

	_, err := p.doRequest(ctx, "UpdateDomainRecord", params)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *AliyunProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	if strings.HasPrefix(zoneID, "esa:") {
		_, err := p.doESARequest(ctx, "DeleteRecord", map[string]string{
			"RecordId": recordID,
		})
		return err
	}

	_, err := p.doRequest(ctx, "DeleteDomainRecord", map[string]string{
		"RecordId": recordID,
	})
	return err
}
