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

const hwEndpoint = "https://dns.myhuaweicloud.com/v2"

type HuaweiProvider struct {
	client    *http.Client
	apiToken  string
	accountID string
}

func NewHuaweiProvider(client *http.Client, token, accountID string) *HuaweiProvider {
	return &HuaweiProvider{
		client:    client,
		apiToken:  token,
		accountID: accountID,
	}
}

func (p *HuaweiProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", hwEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", p.apiToken)
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
		return nil, fmt.Errorf("华为云 DNS 错误 [%d]: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func (p *HuaweiProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, http.MethodGet, "/zones?limit=1", nil)
	return err
}

type hwZoneResp struct {
	Zones []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"zones"`
}

func (p *HuaweiProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, http.MethodGet, "/zones?limit=100", nil)
	if err != nil {
		return nil, err
	}

	var parsed hwZoneResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(parsed.Zones))
	for i, z := range parsed.Zones {
		zones[i] = Zone{
			ID:        z.ID,
			Name:      strings.TrimSuffix(z.Name, "."),
			Status:    z.Status,
			Provider:  "huawei",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type hwRecordResp struct {
	Recordsets []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		TTL         int      `json:"ttl"`
		Records     []string `json:"records"`
		Description string   `json:"description"`
	} `json:"recordsets"`
}

func (p *HuaweiProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	body, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/zones/%s/recordsets?limit=300", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var parsed hwRecordResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	var records []Record
	for _, rs := range parsed.Recordsets {
		cleanName := strings.TrimSuffix(rs.Name, ".")
		displayName := cleanName
		if displayName == zoneName {
			displayName = "@"
		} else if strings.HasSuffix(displayName, "."+zoneName) {
			displayName = strings.TrimSuffix(displayName, "."+zoneName)
		}

		content := strings.Join(rs.Records, ", ")
		records = append(records, Record{
			ID:       rs.ID,
			ZoneID:   zoneID,
			ZoneName: zoneName,
			Type:     rs.Type,
			Name:     displayName,
			Content:  content,
			TTL:      rs.TTL,
			Comment:  rs.Description,
		})
	}
	return records, nil
}

func (p *HuaweiProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	fullRecordName := r.Name
	if fullRecordName == "@" {
		fullRecordName = zoneName + "."
	} else if !strings.HasSuffix(fullRecordName, ".") {
		fullRecordName = fmt.Sprintf("%s.%s.", r.Name, zoneName)
	}

	payload := map[string]any{
		"name":        fullRecordName,
		"type":        r.Type,
		"ttl":         r.TTL,
		"records":     []string{r.Content},
		"description": r.Comment,
	}

	body, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/zones/%s/recordsets", zoneID), payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	r.ID = resp.ID
	r.ZoneID = zoneID
	r.ZoneName = zoneName
	return &r, nil
}

func (p *HuaweiProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	payload := map[string]any{
		"records":     []string{r.Content},
		"ttl":         r.TTL,
		"description": r.Comment,
	}

	_, err := p.doRequest(ctx, http.MethodPut, fmt.Sprintf("/zones/%s/recordsets/%s", zoneID, r.ID), payload)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *HuaweiProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	_, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/zones/%s/recordsets/%s", zoneID, recordID), nil)
	return err
}
