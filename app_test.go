package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"anyzone/internal/core/config"
	"anyzone/internal/core/security"
	"anyzone/internal/core/store"
)

func newTestApp(t *testing.T) (*App, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "anyzone.json")
	s, err := store.NewStoreAt(configPath)
	if err != nil {
		t.Fatalf("NewStoreAt failed: %v", err)
	}
	return &App{
		store: s,
		sec:   security.NewSecurityManager(),
	}, configPath
}

func TestChangePinReEncryptsAccountCredentials(t *testing.T) {
	app, configPath := newTestApp(t)

	if err := app.SetupPin("1234"); err != nil {
		t.Fatalf("SetupPin failed: %v", err)
	}

	credsJSON := `{"api_token":"cf-secret-token-123"}`
	acc, err := app.AddAccount("TestCF", config.ProviderCloudflare, credsJSON, "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// 修改为新的 6 位 PIN
	if err := app.ChangePin("1234", "654321"); err != nil {
		t.Fatalf("ChangePin failed: %v", err)
	}

	// 模拟重新启动应用：从磁盘重新加载 store 并用新 PIN 解锁
	reloadedStore, err := store.NewStoreAt(configPath)
	if err != nil {
		t.Fatalf("reload store failed: %v", err)
	}
	reloadedApp := &App{
		store: reloadedStore,
		sec:   security.NewSecurityManager(),
	}

	if err := reloadedApp.VerifyPin("1234"); err == nil {
		t.Fatalf("expected old PIN to fail after ChangePin")
	}
	if err := reloadedApp.VerifyPin("654321"); err != nil {
		t.Fatalf("VerifyPin with new PIN failed: %v", err)
	}

	detail, err := reloadedApp.GetAccountDetail(acc.ID)
	if err != nil {
		t.Fatalf("GetAccountDetail after ChangePin failed: %v", err)
	}

	creds, ok := detail["credentials"].(config.Credentials)
	if !ok {
		t.Fatalf("unexpected credentials type: %T", detail["credentials"])
	}
	if creds.ApiToken != "cf-secret-token-123" {
		t.Fatalf("expected ApiToken %q, got %q", "cf-secret-token-123", creds.ApiToken)
	}
	if got := reloadedApp.GetSettings().PinLength; got != 6 {
		t.Fatalf("expected PinLength 6, got %d", got)
	}
}

func TestChangePinWrongOldPinPreservesExistingData(t *testing.T) {
	app, _ := newTestApp(t)

	if err := app.SetupPin("1234"); err != nil {
		t.Fatalf("SetupPin failed: %v", err)
	}

	credsJSON := `{"api_token":"preserved-token"}`
	acc, err := app.AddAccount("TestCF", config.ProviderCloudflare, credsJSON, "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	if err := app.ChangePin("9999", "5678"); err == nil {
		t.Fatalf("expected ChangePin with wrong old PIN to fail")
	}

	// 锁定并确认原 PIN 仍可解锁并读取账号凭据
	app.Lock()
	if err := app.VerifyPin("1234"); err != nil {
		t.Fatalf("VerifyPin with original PIN failed after rejected ChangePin: %v", err)
	}
	detail, err := app.GetAccountDetail(acc.ID)
	if err != nil {
		t.Fatalf("GetAccountDetail failed: %v", err)
	}
	creds := detail["credentials"].(config.Credentials)
	if creds.ApiToken != "preserved-token" {
		t.Fatalf("expected ApiToken %q, got %q", "preserved-token", creds.ApiToken)
	}
}

func TestSetupPinRejectsOverwriteWhenAlreadyConfigured(t *testing.T) {
	app, _ := newTestApp(t)

	if err := app.SetupPin("1234"); err != nil {
		t.Fatalf("SetupPin failed: %v", err)
	}
	if err := app.SetupPin("5678"); err == nil {
		t.Fatalf("expected SetupPin to fail when PIN is already configured")
	}
}

func TestListAllZonesReturnsNonNilSliceAndErrorOnAllDecryptionFailure(t *testing.T) {
	app, _ := newTestApp(t)

	if err := app.SetupPin("1234"); err != nil {
		t.Fatalf("SetupPin failed: %v", err)
	}

	// 注入一个密文损坏的账号，模拟全部账号解密失败的场景
	corrupted := config.Account{
		ID:                   "acc_corrupted",
		Name:                 "BrokenAccount",
		Provider:             config.ProviderCloudflare,
		EncryptedCredentials: "not-valid-cipher",
	}
	if err := app.store.AddAccount(corrupted); err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	zones, err := app.ListAllZones()
	if err == nil {
		t.Fatalf("expected ListAllZones to return error when all accounts fail decryption")
	}
	if zones == nil {
		t.Fatalf("expected ListAllZones to return non-nil slice, got nil")
	}
	raw, marshalErr := json.Marshal(zones)
	if marshalErr != nil {
		t.Fatalf("json.Marshal(zones) failed: %v", marshalErr)
	}
	if string(raw) != "[]" {
		t.Fatalf("expected JSON [], got %s", string(raw))
	}
}
