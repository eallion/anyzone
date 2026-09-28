package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const cfApiBase = "https://api.cloudflare.com/client/v4"

type CloudflareProvider struct {
	client    *http.Client
	apiToken  string
	accountID string
}

func NewCloudflareProvider(client *http.Client, apiToken string, accountID string) *CloudflareProvider {
	return &CloudflareProvider{
		client:    client,
		apiToken:  apiToken,
		accountID: accountID,
	}
}

type cfResponse struct {
	Success  bool            `json:"success"`
	Errors   []cfError       `json:"errors"`
	Messages []string        `json:"messages"`
	Result   json.RawMessage `json:"result"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (p *CloudflareProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfApiBase+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed cfResponse
	if err := json.Unmarshal(respData, &parsed); err != nil {
		return nil, fmt.Errorf("解析 Cloudflare 返回数据失败: %w", err)
	}

	if !parsed.Success {
		var errMsgs []string
		for _, e := range parsed.Errors {
			errMsgs = append(errMsgs, fmt.Sprintf("[%d] %s", e.Code, e.Message))
		}
		return nil, fmt.Errorf("Cloudflare 错误: %s", strings.Join(errMsgs, "; "))
	}

	return parsed.Result, nil
}

func (p *CloudflareProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, http.MethodGet, "/user/tokens/verify", nil)
	return err
}

type cfZoneItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (p *CloudflareProvider) ListZones(ctx context.Context) ([]Zone, error) {
	raw, err := p.doRequest(ctx, http.MethodGet, "/zones?per_page=50", nil)
	if err != nil {
		return nil, err
	}

	var items []cfZoneItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(items))
	for i, it := range items {
		zones[i] = Zone{
			ID:        it.ID,
			Name:      it.Name,
			Status:    it.Status,
			Provider:  "cloudflare",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type cfRecordItem struct {
	ID       string `json:"id"`
	ZoneID   string `json:"zone_id"`
	ZoneName string `json:"zone_name"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Proxied  bool   `json:"proxied"`
	Comment  string `json:"comment,omitempty"`
}

func (p *CloudflareProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	raw, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/zones/%s/dns_records?per_page=100", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var items []cfRecordItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}

	records := make([]Record, len(items))
	for i, it := range items {
		proxiedVal := it.Proxied
		displayName := it.Name
		if displayName == zoneName {
			displayName = "@"
		} else if strings.HasSuffix(displayName, "."+zoneName) {
			displayName = strings.TrimSuffix(displayName, "."+zoneName)
		}

		records[i] = Record{
			ID:       it.ID,
			ZoneID:   it.ZoneID,
			ZoneName: it.ZoneName,
			Type:     it.Type,
			Name:     displayName,
			Content:  it.Content,
			TTL:      it.TTL,
			Priority: it.Priority,
			Proxied:  &proxiedVal,
			Comment:  it.Comment,
		}
	}
	return records, nil
}

func (p *CloudflareProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	fullName := r.Name
	if fullName == "@" {
		fullName = zoneName
	} else if !strings.HasSuffix(fullName, zoneName) {
		fullName = fmt.Sprintf("%s.%s", r.Name, zoneName)
	}

	payload := map[string]any{
		"type":    r.Type,
		"name":    fullName,
		"content": r.Content,
		"ttl":     r.TTL,
	}
	if r.Proxied != nil {
		payload["proxied"] = *r.Proxied
	}
	if r.Priority > 0 {
		payload["priority"] = r.Priority
	}
	if r.Comment != "" {
		payload["comment"] = r.Comment
	}

	raw, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/zones/%s/dns_records", zoneID), payload)
	if err != nil {
		return nil, err
	}

	var created cfRecordItem
	if err := json.Unmarshal(raw, &created); err != nil {
		return nil, err
	}

	proxiedVal := created.Proxied
	return &Record{
		ID:       created.ID,
		ZoneID:   created.ZoneID,
		ZoneName: created.ZoneName,
		Type:     created.Type,
		Name:     r.Name,
		Content:  created.Content,
		TTL:      created.TTL,
		Priority: created.Priority,
		Proxied:  &proxiedVal,
		Comment:  created.Comment,
	}, nil
}

func (p *CloudflareProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	fullName := r.Name
	if fullName == "@" {
		fullName = zoneName
	} else if !strings.HasSuffix(fullName, zoneName) {
		fullName = fmt.Sprintf("%s.%s", r.Name, zoneName)
	}

	payload := map[string]any{
		"type":    r.Type,
		"name":    fullName,
		"content": r.Content,
		"ttl":     r.TTL,
	}
	if r.Proxied != nil {
		payload["proxied"] = *r.Proxied
	}
	if r.Priority > 0 {
		payload["priority"] = r.Priority
	}
	if r.Comment != "" {
		payload["comment"] = r.Comment
	}

	raw, err := p.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, r.ID), payload)
	if err != nil {
		return nil, err
	}

	var updated cfRecordItem
	if err := json.Unmarshal(raw, &updated); err != nil {
		return nil, err
	}

	proxiedVal := updated.Proxied
	return &Record{
		ID:       updated.ID,
		ZoneID:   updated.ZoneID,
		ZoneName: updated.ZoneName,
		Type:     updated.Type,
		Name:     r.Name,
		Content:  updated.Content,
		TTL:      updated.TTL,
		Priority: updated.Priority,
		Proxied:  &proxiedVal,
		Comment:  updated.Comment,
	}, nil
}

func (p *CloudflareProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	_, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID), nil)
	return err
}
