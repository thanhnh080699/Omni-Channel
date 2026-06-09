package handlers

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWhatsAppQRCacheReturnsCachedQRInsideFetchWindow(t *testing.T) {
	handler := &Handler{qrCache: &qrCache{entries: map[string]qrCacheEntry{}}}
	expiresAt := time.Now().UTC().Add(20 * time.Second).Format(time.RFC3339Nano)
	body := []byte(`{"accountId":"ca_1","status":"qr","qr":"qr-token","qrExpiresAt":"` + expiresAt + `"}`)

	merged := handler.mergeWhatsAppQRCache("usr_1:ca_1", body)
	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["qr"] != "qr-token" || payload["cached"] != false {
		t.Fatalf("unexpected merged payload: %#v", payload)
	}
	if payload["qr_expires_at"] == nil {
		t.Fatalf("expected qr_expires_at in merged payload: %#v", payload)
	}

	cached, ok := handler.cachedWhatsAppQR("usr_1:ca_1")
	if !ok {
		t.Fatal("expected cached QR")
	}
	if cached["qr"] != "qr-token" || cached["cached"] != true {
		t.Fatalf("unexpected cached payload: %#v", cached)
	}
	if cached["qr_expires_at"] == nil {
		t.Fatalf("expected qr_expires_at in cached payload: %#v", cached)
	}
}

func TestWhatsAppQRCacheExpiresFetchWindow(t *testing.T) {
	handler := &Handler{qrCache: &qrCache{entries: map[string]qrCacheEntry{
		"usr_1:ca_1": {QR: "qr-token", Status: "qr", UpdatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(-time.Second)},
	}}}
	cached, ok := handler.cachedWhatsAppQR("usr_1:ca_1")
	if ok || cached != nil {
		t.Fatalf("expected cache miss after fetch window, got %#v", cached)
	}
}

func TestWhatsAppQRCacheReturnsAccountID(t *testing.T) {
	handler := &Handler{qrCache: &qrCache{entries: map[string]qrCacheEntry{
		"usr_1:ca_1": {QR: "qr-token", Status: "qr", UpdatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Minute)},
	}}}
	cached, ok := handler.cachedWhatsAppQR("usr_1:ca_1")
	if !ok {
		t.Fatal("expected cached QR")
	}
	if cached["accountId"] != "ca_1" {
		t.Fatalf("expected accountId ca_1, got %#v", cached["accountId"])
	}
}

func TestWhatsAppQRCacheRejectsExpiredAdapterQR(t *testing.T) {
	handler := &Handler{qrCache: &qrCache{entries: map[string]qrCacheEntry{}}}
	expiredAt := time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)
	body := []byte(`{"accountId":"ca_1","status":"qr","qr":"qr-token","qrExpiresAt":"` + expiredAt + `"}`)

	merged := handler.mergeWhatsAppQRCache("usr_1:ca_1", body)

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["qr"] != nil {
		t.Fatalf("expected expired QR to be removed, got %#v", payload)
	}
	if payload["status"] != "connecting" {
		t.Fatalf("expected connecting status after expired QR, got %#v", payload)
	}
	if _, ok := handler.cachedWhatsAppQR("usr_1:ca_1"); ok {
		t.Fatal("expected expired QR to be absent from cache")
	}
}
