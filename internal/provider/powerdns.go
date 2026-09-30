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

type PowerDNSProvider struct {
	client    *http.Client
	serverURL string
	apiKey    string
	accountID string
}

func NewPowerDNSProvider(client *http.Client, serverURL, apiKey, accountID string) *PowerDNSProvider {
	trimmed := strings.TrimRight(serverURL, "/")
	if !strings.HasSuffix(trimmed, "/api/v1") {
		trimmed = trimmed + "/api/v1"
	}
	return &PowerDNSProvider{
		client:    client,
		serverURL: trimmed,
		apiKey:    apiKey,
		accountID: accountID,
	}
}

func (p *PowerDNSProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", p.serverURL, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", p.apiKey)
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
		return nil, fmt.Errorf("PowerDNS 错误 [%d]: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func (p *PowerDNSProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, http.MethodGet, "/servers/localhost", nil)
	return err
}

type pdnsZoneItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func (p *PowerDNSProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, http.MethodGet, "/servers/localhost/zones", nil)
	if err != nil {
		return nil, err
	}

	var items []pdnsZoneItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(items))
	for i, z := range items {
		cleanName := strings.TrimSuffix(z.Name, ".")
		zones[i] = Zone{
			ID:        z.ID,
			Name:      cleanName,
			Status:    z.Kind,
			Provider:  "powerdns",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type pdnsZoneDetailResp struct {
	RRsets []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		TTL     int    `json:"ttl"`
		Records []struct {
			Content  string `json:"content"`
			Disabled bool   `json:"disabled"`
		} `json:"records"`
	} `json:"rrsets"`
}

func (p *PowerDNSProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	body, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/servers/localhost/zones/%s", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var parsed pdnsZoneDetailResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	var records []Record
	for _, rr := range parsed.RRsets {
		cleanRRName := strings.TrimSuffix(rr.Name, ".")
		displayName := cleanRRName
		if displayName == zoneName {
			displayName = "@"
		} else if strings.HasSuffix(displayName, "."+zoneName) {
			displayName = strings.TrimSuffix(displayName, "."+zoneName)
		}

		for _, rec := range rr.Records {
			id := fmt.Sprintf("%s:%s:%s", rr.Type, cleanRRName, rec.Content)
			records = append(records, Record{
				ID:       id,
				ZoneID:   zoneID,
				ZoneName: zoneName,
				Type:     rr.Type,
				Name:     displayName,
				Content:  rec.Content,
				TTL:      rr.TTL,
			})
		}
	}
	return records, nil
}

func (p *PowerDNSProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	fullRecordName := r.Name
	if fullRecordName == "@" {
		fullRecordName = zoneName + "."
	} else if !strings.HasSuffix(fullRecordName, ".") {
		fullRecordName = fmt.Sprintf("%s.%s.", r.Name, zoneName)
	}

	payload := map[string]any{
		"rrsets": []map[string]any{
			{
				"name":       fullRecordName,
				"type":       r.Type,
				"ttl":        r.TTL,
				"changetype": "REPLACE",
				"records": []map[string]any{
					{
						"content":  r.Content,
						"disabled": false,
					},
				},
			},
		},
	}

	_, err := p.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/servers/localhost/zones/%s", zoneID), payload)
	if err != nil {
		return nil, err
	}

	r.ID = fmt.Sprintf("%s:%s:%s", r.Type, strings.TrimSuffix(fullRecordName, "."), r.Content)
	r.ZoneID = zoneID
	r.ZoneName = zoneName
	return &r, nil
}

func (p *PowerDNSProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	return p.CreateRecord(ctx, zoneID, zoneName, r)
}

func (p *PowerDNSProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	parts := strings.Split(recordID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("非法记录标识")
	}

	recType := parts[0]
	recName := parts[1]
	if !strings.HasSuffix(recName, ".") {
		recName += "."
	}

	payload := map[string]any{
		"rrsets": []map[string]any{
			{
				"name":       recName,
				"type":       recType,
				"changetype": "DELETE",
			},
		},
	}

	_, err := p.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/servers/localhost/zones/%s", zoneID), payload)
	return err
}
