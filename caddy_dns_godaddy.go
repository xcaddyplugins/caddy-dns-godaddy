package caddy_dns_godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/libdns/libdns"
)

func init() {
	caddy.RegisterModule(Provider{})
}

// Provider implements the libdns interfaces for GoDaddy DNS
type Provider struct {
	// GoDaddy API Key
	APIKey string `json:"api_key,omitempty"`
	// GoDaddy API Secret
	APISecret string `json:"api_secret,omitempty"`
	// HTTP request timeout
	HTTPTimeout caddy.Duration `json:"http_timeout,omitempty"`
}

// CaddyModule returns the Caddy module information
func (Provider) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "dns.providers.godaddy",
		New: func() caddy.Module { return &Provider{} },
	}
}

// Provision sets up the provider
func (p *Provider) Provision(ctx caddy.Context) error {
	if p.APIKey == "" {
		return fmt.Errorf("GoDaddy API key cannot be empty")
	}
	if p.APISecret == "" {
		return fmt.Errorf("GoDaddy API secret cannot be empty")
	}
	if p.HTTPTimeout == 0 {
		p.HTTPTimeout = caddy.Duration(30 * time.Second)
	}
	return nil
}

// UnmarshalCaddyfile parses Caddyfile configuration
func (p *Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "api_key":
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.APIKey = d.Val()
			case "api_secret":
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.APISecret = d.Val()
			case "http_timeout":
				if !d.NextArg() {
					return d.ArgErr()
				}
				dur, err := time.ParseDuration(d.Val())
				if err != nil {
					return fmt.Errorf("failed to parse http_timeout: %v", err)
				}
				p.HTTPTimeout = caddy.Duration(dur)
			default:
				return d.Errf("unknown configuration option: %s", d.Val())
			}
		}
	}
	return nil
}

func (p *Provider) request(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", p.APIKey, p.APISecret))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(p.HTTPTimeout)}
	return client.Do(req)
}

// AppendRecords adds DNS records to the domain
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var createdRecords []libdns.Record

	for _, record := range records {
		err := p.createRecord(ctx, zone, record)
		if err != nil {
			return createdRecords, fmt.Errorf("failed to create DNS record: %v", err)
		}
		createdRecords = append(createdRecords, record)
	}

	return createdRecords, nil
}

// DeleteRecords deletes DNS records from the domain
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deletedRecords []libdns.Record

	for _, record := range records {
		err := p.deleteRecord(ctx, zone, record)
		if err != nil {
			return deletedRecords, fmt.Errorf("failed to delete DNS record: %v", err)
		}
		deletedRecords = append(deletedRecords, record)
	}

	return deletedRecords, nil
}

// GetRecords retrieves DNS records for the domain
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	domain := strings.TrimSuffix(zone, ".")

	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records", domain)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.request(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GoDaddy API request failed: %d %s", resp.StatusCode, string(body))
	}

	var godaddyRecords []libdns.Record
	if err := json.NewDecoder(resp.Body).Decode(&godaddyRecords); err != nil {
		return nil, err
	}

	var records []libdns.Record
	for _, gr := range godaddyRecords {
		rr := gr.RR()
		records = append(records, libdns.RR{
			Type: rr.Type,
			Name: rr.Name,
			Data: rr.Data,
			TTL:  rr.TTL,
		})
	}

	return records, nil
}

// SetRecords sets DNS records for the domain
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// First get existing records
	existingRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	// Delete records to be replaced
	for _, newRecord := range records {
		for _, existing := range existingRecords {
			existingRR := existing.RR()
			newRecordRR := newRecord.RR()
			if existingRR.Type == newRecordRR.Type && existingRR.Name == newRecordRR.Name {
				err := p.deleteRecord(ctx, zone, existing)
				if err != nil {
					return nil, fmt.Errorf("failed to delete existing record: %v", err)
				}
			}
		}
	}

	// Add new records
	return p.AppendRecords(ctx, zone, records)
}

// createRecord creates a single DNS record
func (p *Provider) createRecord(ctx context.Context, zone string, record libdns.Record) error {
	domain := strings.TrimSuffix(zone, ".")
	recordName := strings.TrimSuffix(record.RR().Name, "."+domain)
	if recordName == domain {
		recordName = "@"
	}
	rr := record.RR()
	godaddyRecord := libdns.RR{
		Type: rr.Type,
		Name: recordName,
		Data: rr.Data,
		TTL:  rr.TTL,
	}

	recordsArray := []libdns.RR{godaddyRecord}
	jsonData, err := json.Marshal(recordsArray)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records", domain)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	resp, err := p.request(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create DNS record: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// deleteRecord deletes a single DNS record
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) error {
	domain := strings.TrimSuffix(zone, ".")
	rr := record.RR()
	recordName := strings.TrimSuffix(rr.Name, "."+domain)
	if recordName == domain {
		recordName = "@"
	}

	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/%s/%s",
		domain, rr.Type, recordName)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.request(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete DNS record: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
	_ caddy.Provisioner     = (*Provider)(nil)
	_ caddyfile.Unmarshaler = (*Provider)(nil)
)
