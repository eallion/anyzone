package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const hetznerEndpoint = "https://dns.hetzner.com/api/v1"

type HetznerProvider struct {
	client    *http.Client
	apiToken  string
	accountID string
}

func NewHetznerProvider(client *http.Client, token, accountID string) *HetznerProvider {
	return &HetznerProvider{
		client:    client,
		apiToken:  token,
		accountID: accountID,
	}
}

func (p *HetznerProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", hetznerEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Auth-API-Token", p.apiToken)
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
		return nil, fmt.Errorf("Hetzner 错误 [%d]: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func (p *HetznerProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, http.MethodGet, "/zones?per_page=1", nil)
	return err
}

type hetznerZoneResp struct {
	Zones []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"zones"`
}

func (p *HetznerProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, http.MethodGet, "/zones", nil)
	if err != nil {
		return nil, err
	}

	var parsed hetznerZoneResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(parsed.Zones))
	for i, z := range parsed.Zones {
		zones[i] = Zone{
			ID:        z.ID,
			Name:      z.Name,
			Status:    z.Status,
			Provider:  "hetzner",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type hetznerRecordResp struct {
	Records []struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Name   string `json:"name"`
		Value  string `json:"value"`
		TTL    int    `json:"ttl"`
		ZoneID string `json:"zone_id"`
	} `json:"records"`
}

func (p *HetznerProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	body, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/records?zone_id=%s", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var parsed hetznerRecordResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	records := make([]Record, len(parsed.Records))
	for i, r := range parsed.Records {
		name := r.Name
		if name == "" {
			name = "@"
		}
		records[i] = Record{
			ID:       r.ID,
			ZoneID:   zoneID,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     name,
			Content:  r.Value,
			TTL:      r.TTL,
		}
	}
	return records, nil
}

func (p *HetznerProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	name := r.Name
	if name == "@" {
		name = ""
	}

	payload := map[string]any{
		"zone_id": zoneID,
		"type":    r.Type,
		"name":    name,
		"value":   r.Content,
		"ttl":     r.TTL,
	}

	body, err := p.doRequest(ctx, http.MethodPost, "/records", payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	r.ID = resp.Record.ID
	r.ZoneID = zoneID
	r.ZoneName = zoneName
	return &r, nil
}

func (p *HetznerProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	name := r.Name
	if name == "@" {
		name = ""
	}

	payload := map[string]any{
		"zone_id": zoneID,
		"type":    r.Type,
		"name":    name,
		"value":   r.Content,
		"ttl":     r.TTL,
	}

	_, err := p.doRequest(ctx, http.MethodPut, fmt.Sprintf("/records/%s", r.ID), payload)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *HetznerProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	_, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/records/%s", recordID), nil)
	return err
}
