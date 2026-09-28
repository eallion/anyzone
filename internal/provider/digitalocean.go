package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const doEndpoint = "https://api.digitalocean.com/v2"

type DigitalOceanProvider struct {
	client    *http.Client
	apiToken  string
	accountID string
}

func NewDigitalOceanProvider(client *http.Client, token, accountID string) *DigitalOceanProvider {
	return &DigitalOceanProvider{
		client:    client,
		apiToken:  token,
		accountID: accountID,
	}
}

func (p *DigitalOceanProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", doEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DigitalOcean 错误 [%d]: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func (p *DigitalOceanProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, http.MethodGet, "/domains?per_page=1", nil)
	return err
}

type doDomainListResp struct {
	Domains []struct {
		Name string `json:"name"`
		TTL  int    `json:"ttl"`
	} `json:"domains"`
}

func (p *DigitalOceanProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, http.MethodGet, "/domains?per_page=100", nil)
	if err != nil {
		return nil, err
	}

	var parsed doDomainListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(parsed.Domains))
	for i, d := range parsed.Domains {
		zones[i] = Zone{
			ID:        d.Name,
			Name:      d.Name,
			Status:    "active",
			Provider:  "digitalocean",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type doRecordListResp struct {
	DomainRecords []struct {
		ID       int64  `json:"id"`
		Type     string `json:"type"`
		Name     string `json:"name"`
		Data     string `json:"data"`
		Priority *int   `json:"priority"`
		TTL      int    `json:"ttl"`
	} `json:"domain_records"`
}

func (p *DigitalOceanProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	body, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/domains/%s/records?per_page=200", zoneName), nil)
	if err != nil {
		return nil, err
	}

	var parsed doRecordListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	records := make([]Record, len(parsed.DomainRecords))
	for i, r := range parsed.DomainRecords {
		prio := 0
		if r.Priority != nil {
			prio = *r.Priority
		}
		records[i] = Record{
			ID:       strconv.FormatInt(r.ID, 10),
			ZoneID:   zoneName,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     r.Name,
			Content:  r.Data,
			TTL:      r.TTL,
			Priority: prio,
		}
	}
	return records, nil
}

func (p *DigitalOceanProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	payload := map[string]any{
		"type": r.Type,
		"name": r.Name,
		"data": r.Content,
		"ttl":  r.TTL,
	}
	if r.Priority > 0 {
		payload["priority"] = r.Priority
	}

	body, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/domains/%s/records", zoneName), payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		DomainRecord struct {
			ID int64 `json:"id"`
		} `json:"domain_record"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	r.ID = strconv.FormatInt(resp.DomainRecord.ID, 10)
	r.ZoneID = zoneName
	r.ZoneName = zoneName
	return &r, nil
}

func (p *DigitalOceanProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	payload := map[string]any{
		"type": r.Type,
		"name": r.Name,
		"data": r.Content,
		"ttl":  r.TTL,
	}
	if r.Priority > 0 {
		payload["priority"] = r.Priority
	}

	_, err := p.doRequest(ctx, http.MethodPut, fmt.Sprintf("/domains/%s/records/%s", zoneName, r.ID), payload)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *DigitalOceanProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	_, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/domains/%s/records/%s", zoneName, recordID), nil)
	return err
}
