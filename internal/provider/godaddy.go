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

const godaddyEndpoint = "https://api.godaddy.com/v1"

type GoDaddyProvider struct {
	client    *http.Client
	apiKey    string
	apiSecret string
	accountID string
}

func NewGoDaddyProvider(client *http.Client, key, secret, accountID string) *GoDaddyProvider {
	return &GoDaddyProvider{
		client:    client,
		apiKey:    key,
		apiSecret: secret,
		accountID: accountID,
	}
}

func (p *GoDaddyProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", godaddyEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", p.apiKey, p.apiSecret))
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GoDaddy 错误 [%d]: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func (p *GoDaddyProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, http.MethodGet, "/domains?limit=1", nil)
	return err
}

type gdDomainItem struct {
	Domain string `json:"domain"`
	Status string `json:"status"`
}

func (p *GoDaddyProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, http.MethodGet, "/domains?limit=100", nil)
	if err != nil {
		return nil, err
	}

	var items []gdDomainItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(items))
	for i, d := range items {
		zones[i] = Zone{
			ID:        d.Domain,
			Name:      d.Domain,
			Status:    d.Status,
			Provider:  "godaddy",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type gdRecordItem struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

func (p *GoDaddyProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	body, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/domains/%s/records", zoneName), nil)
	if err != nil {
		return nil, err
	}

	var items []gdRecordItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	records := make([]Record, len(items))
	for i, r := range items {
		// GoDaddy API 无专属记录 ID，以 type:name 组成逻辑标识
		id := fmt.Sprintf("%s:%s", r.Type, r.Name)
		records[i] = Record{
			ID:       id,
			ZoneID:   zoneName,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     r.Name,
			Content:  r.Data,
			TTL:      r.TTL,
			Priority: r.Priority,
		}
	}
	return records, nil
}

func (p *GoDaddyProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	payload := []gdRecordItem{
		{
			Type:     r.Type,
			Name:     r.Name,
			Data:     r.Content,
			TTL:      r.TTL,
			Priority: r.Priority,
		},
	}

	_, err := p.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/domains/%s/records", zoneName), payload)
	if err != nil {
		return nil, err
	}

	r.ID = fmt.Sprintf("%s:%s", r.Type, r.Name)
	r.ZoneID = zoneName
	r.ZoneName = zoneName
	return &r, nil
}

func (p *GoDaddyProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	payload := []gdRecordItem{
		{
			Data:     r.Content,
			TTL:      r.TTL,
			Priority: r.Priority,
		},
	}

	parts := strings.SplitN(r.ID, ":", 2)
	recType := r.Type
	recName := r.Name
	if len(parts) == 2 {
		recType = parts[0]
		recName = parts[1]
	}

	_, err := p.doRequest(ctx, http.MethodPut, fmt.Sprintf("/domains/%s/records/%s/%s", zoneName, recType, recName), payload)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *GoDaddyProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	parts := strings.SplitN(recordID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("非法记录标识")
	}

	_, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/domains/%s/records/%s/%s", zoneName, parts[0], parts[1]), nil)
	return err
}
