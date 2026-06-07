package handlers

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWhatsAppQRCacheReturnsCachedQRInsideFetchWindow(t *testing.T) {
	handler := &Handler{qrCache: &qrCache{entries: map[string]qrCacheEntry{}}}
	body := []byte(`{"accountId":"ca_1","status":"qr","qr":"qr-token"}`)

	merged := handler.mergeWhatsAppQRCache("usr_1:ca_1", body)
	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["qr"] != "qr-token" || payload["cached"] != false {
		t.Fatalf("unexpected merged payload: %#v", payload)
	}

	cached, ok := handler.cachedWhatsAppQR("usr_1:ca_1")
	if !ok {
		t.Fatal("expected cached QR")
	}
	if cached["qr"] != "qr-token" || cached["cached"] != true {
		t.Fatalf("unexpected cached payload: %#v", cached)
	}
}

func TestWhatsAppQRCacheExpiresFetchWindow(t *testing.T) {
	handler := &Handler{qrCache: &qrCache{entries: map[string]qrCacheEntry{
		"usr_1:ca_1": {QR: "qr-token", Status: "qr", UpdatedAt: time.Now().UTC(), FetchAfter: time.Now().UTC().Add(-time.Second)},
	}}}
	cached, ok := handler.cachedWhatsAppQR("usr_1:ca_1")
	if ok || cached != nil {
		t.Fatalf("expected cache miss after fetch window, got %#v", cached)
	}
}
