package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const pbEndpoint = "https://api.porkbun.com/api/json/v3"

type PorkbunProvider struct {
	client       *http.Client
	apiKey       string
	secretApiKey string
	accountID    string
}

func NewPorkbunProvider(client *http.Client, apiKey, secretApiKey, accountID string) *PorkbunProvider {
	return &PorkbunProvider{
		client:       client,
		apiKey:       apiKey,
		secretApiKey: secretApiKey,
		accountID:    accountID,
	}
}

type pbBaseResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (p *PorkbunProvider) doRequest(ctx context.Context, path string, extraPayload map[string]any) ([]byte, error) {
	payload := map[string]any{
		"apikey":       p.apiKey,
		"secretapikey": p.secretApiKey,
	}
	for k, v := range extraPayload {
		payload[k] = v
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", pbEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var base pbBaseResp
	if err := json.Unmarshal(body, &base); err == nil && base.Status == "ERROR" {
		return nil, fmt.Errorf("Porkbun 错误: %s", base.Message)
	}

	return body, nil
}

func (p *PorkbunProvider) TestConnection(ctx context.Context) error {
	_, err := p.doRequest(ctx, "/ping", nil)
	return err
}

type pbDomainListResp struct {
	pbBaseResp
	Domains []struct {
		Domain string `json:"domain"`
		Status string `json:"status"`
	} `json:"domains"`
}

func (p *PorkbunProvider) ListZones(ctx context.Context) ([]Zone, error) {
	body, err := p.doRequest(ctx, "/domain/listAll", nil)
	if err != nil {
		return nil, err
	}

	var parsed pbDomainListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	zones := make([]Zone, len(parsed.Domains))
	for i, d := range parsed.Domains {
		zones[i] = Zone{
			ID:        d.Domain,
			Name:      d.Domain,
			Status:    d.Status,
			Provider:  "porkbun",
			AccountID: p.accountID,
		}
	}
	return zones, nil
}

type pbRecordListResp struct {
	pbBaseResp
	Records []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
		TTL     string `json:"ttl"`
		Prio    string `json:"prio"`
		Notes   string `json:"notes"`
	} `json:"records"`
}

func (p *PorkbunProvider) ListRecords(ctx context.Context, zoneID string, zoneName string) ([]Record, error) {
	body, err := p.doRequest(ctx, fmt.Sprintf("/dns/retrieve/%s", zoneName), nil)
	if err != nil {
		return nil, err
	}

	var parsed pbRecordListResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	records := make([]Record, len(parsed.Records))
	for i, r := range parsed.Records {
		ttlVal, _ := strconv.Atoi(r.TTL)
		prioVal, _ := strconv.Atoi(r.Prio)

		displayName := r.Name
		if displayName == zoneName {
			displayName = "@"
		} else if strings.HasSuffix(displayName, "."+zoneName) {
			displayName = strings.TrimSuffix(displayName, "."+zoneName)
		}

		records[i] = Record{
			ID:       r.ID,
			ZoneID:   zoneName,
			ZoneName: zoneName,
			Type:     r.Type,
			Name:     displayName,
			Content:  r.Content,
			TTL:      ttlVal,
			Priority: prioVal,
			Comment:  r.Notes,
		}
	}
	return records, nil
}

func (p *PorkbunProvider) CreateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	name := r.Name
	if name == "@" {
		name = ""
	}

	extra := map[string]any{
		"name":    name,
		"type":    r.Type,
		"content": r.Content,
		"ttl":     strconv.Itoa(r.TTL),
	}
	if r.Priority > 0 {
		extra["prio"] = strconv.Itoa(r.Priority)
	}
	if r.Comment != "" {
		extra["notes"] = r.Comment
	}

	body, err := p.doRequest(ctx, fmt.Sprintf("/dns/create/%s", zoneName), extra)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	r.ID = strconv.FormatInt(resp.ID, 10)
	r.ZoneID = zoneName
	r.ZoneName = zoneName
	return &r, nil
}

func (p *PorkbunProvider) UpdateRecord(ctx context.Context, zoneID string, zoneName string, r Record) (*Record, error) {
	name := r.Name
	if name == "@" {
		name = ""
	}

	extra := map[string]any{
		"name":    name,
		"type":    r.Type,
		"content": r.Content,
		"ttl":     strconv.Itoa(r.TTL),
	}
	if r.Priority > 0 {
		extra["prio"] = strconv.Itoa(r.Priority)
	}
	if r.Comment != "" {
		extra["notes"] = r.Comment
	}

	_, err := p.doRequest(ctx, fmt.Sprintf("/dns/edit/%s/%s", zoneName, r.ID), extra)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (p *PorkbunProvider) DeleteRecord(ctx context.Context, zoneID string, zoneName string, recordID string) error {
	_, err := p.doRequest(ctx, fmt.Sprintf("/dns/delete/%s/%s", zoneName, recordID), nil)
	return err
}
