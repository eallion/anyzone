package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const dnspodEndpoint = "https://dnsapi.cn"

type DNSPodProvider struct {
	client        *http.Client
	loginToken    string // 格式: "ID,Token"
	accountID     string
	enableEdgeOne bool
}

func NewDNSPodProvider(client *http.Client, tokenID, token, accountID string, enableEdgeOne bool) *DNSPodProvider {
	return &DNSPodProvider{
		client:        client,
		loginToken:    fmt.Sprintf("%s,%s", tokenID, token),
		accountID:     accountID,
		enableEdgeOne: enableEdgeOne,
	}
}

type dpStatusResp struct {
	Status struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
}

func (p *DNSPodProvider) doRequest(ctx context.Context, action string, extra url.Values) ([]byte, error) {
	form := url.Values{}
	form.Set("login_token", p.loginToken)
	form.Set("format", "json")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	reqURL := fmt.Sprintf("%s/%s", dnspodEndpoint, action)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "AnyZone/1.0.0 (admin@anyzone.local)")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyStr := strings.TrimSpace(string(body))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DNSPod HTTP 状态错误 [%d]: %s", resp.StatusCode, bodyStr)
	}
	if strings.HasPrefix(bodyStr, "<") {
		return nil, fmt.Errorf("DNSPod 接口返回非 JSON 响应 (可能是 Token 错误或接口拦截): %s", bodyStr)
	}

	var statusCheck dpStatusResp
	if err := json.Unmarshal(body, &statusCheck); err == nil {
		if statusCheck.Status.Code != "1" && statusCheck.Status.Code != "" {
			return nil, fmt.Errorf("DNSPod 错误 [%s]: %s", statusCheck.Status.Code, statusCheck.Status.Message)
		}
	}

	return body, nil
}

func (p *DNSPodProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, "User.Detail", nil)
	return err
}

type dpDomainListResp struct {
	Domains []struct {
		ID     any    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"domains"`
}

func (p *DNSPodProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, "Domain.List", nil)
	if err != nil {
		return nil, err
	}

	var parsed dpDomainListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(parsed.Domains))
	for i, d := range parsed.Domains {
		zones[i] = Zone{
			ID:        fmt.Sprintf("%v", d.ID),
			Name:      d.Name,
			Status:    d.Status,
			Provider:  "dnspod",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type dpRecordListResp struct {
	Records []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Value   string `json:"value"`
		TTL     any    `json:"ttl"`
		MX      any    `json:"mx"`
		Remark  string `json:"remark"`
	} `json:"records"`
}

func (p *DNSPodProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	form := url.Values{}
	form.Set("domain_id", zoneID)
	form.Set("length", "3000")

	body, err := p.doRequest(ctx, "Record.List", form)
	if err != nil {
		return nil, err
	}

	var parsed dpRecordListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	records := make([]Record, len(parsed.Records))
	for i, r := range parsed.Records {
		ttlInt, _ := strconv.Atoi(fmt.Sprintf("%v", r.TTL))
		mxInt, _ := strconv.Atoi(fmt.Sprintf("%v", r.MX))

		records[i] = Record{
			ID:       r.ID,
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

func (p *DNSPodProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	form := url.Values{}
	form.Set("domain_id", zoneID)
	form.Set("sub_domain", r.Name)
	form.Set("record_type", r.Type)
	form.Set("record_line", "默认")
	form.Set("value", r.Content)
	if r.TTL > 0 {
		form.Set("ttl", strconv.Itoa(r.TTL))
	}
	if r.Priority > 0 {
		form.Set("mx", strconv.Itoa(r.Priority))
	}

	body, err := p.doRequest(ctx, "Record.Create", form)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Record struct {
			ID any `json:"id"`
		} `json:"record"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	r.ID = fmt.Sprintf("%v", resp.Record.ID)
	r.ZoneID = zoneID
	r.ZoneName = zoneName
	return &r, nil
}

func (p *DNSPodProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	form := url.Values{}
	form.Set("domain_id", zoneID)
	form.Set("record_id", r.ID)
	form.Set("sub_domain", r.Name)
	form.Set("record_type", r.Type)
	form.Set("record_line", "默认")
	form.Set("value", r.Content)
	if r.TTL > 0 {
		form.Set("ttl", strconv.Itoa(r.TTL))
	}
	if r.Priority > 0 {
		form.Set("mx", strconv.Itoa(r.Priority))
	}

	_, err := p.doRequest(ctx, "Record.Modify", form)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *DNSPodProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	form := url.Values{}
	form.Set("domain_id", zoneID)
	form.Set("record_id", recordID)

	_, err := p.doRequest(ctx, "Record.Remove", form)
	return err
}
