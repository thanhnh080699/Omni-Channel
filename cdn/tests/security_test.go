package tests

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"meditour/cdn/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity(t *testing.T) {
	router := SetupTestRouter()
	SetupTestDirs()
	defer CleanupTestDirs()

	createTestImage(t, "./test_uploads/secure.png")

	t.Run("API Key (Missing)", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/folder", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("API Key (Invalid)", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/folder", nil)
		req.Header.Set("X-API-KEY", "wrong_key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Folder Path Traversal (Absolute Windows Path)", func(t *testing.T) {
		body := bytes.NewBufferString(`{"path":"C:\\temp\\escape"}`)
		req, _ := http.NewRequest("POST", "/api/folder", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-KEY", "test_key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Signature (Enabled but Missing)", func(t *testing.T) {
		// Temporarily enable signature requirement
		oldReq := SetupTestRouter() // Get a router with current config

		// In test_utils, we used a pointer to config.AppConfig
		// Let's enable it
		orig := config.AppConfig.RequireSignature
		config.AppConfig.RequireSignature = true
		defer func() { config.AppConfig.RequireSignature = orig }()

		req, _ := http.NewRequest("GET", "/uploads/secure.png", nil)
		w := httptest.NewRecorder()
		oldReq.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Signature (Valid)", func(t *testing.T) {
		orig := config.AppConfig.RequireSignature
		config.AppConfig.RequireSignature = true
		defer func() { config.AppConfig.RequireSignature = orig }()

		path := "/secure.png"
		exp := "1999999999" // Far future
		data := fmt.Sprintf("%s%s", path, exp)
		h := hmac.New(sha256.New, []byte(config.AppConfig.SignatureKey))
		h.Write([]byte(data))
		sig := hex.EncodeToString(h.Sum(nil))

		url := fmt.Sprintf("/uploads%s?sig=%s&exp=%s", path, sig, exp)
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Signature (Expired)", func(t *testing.T) {
		orig := config.AppConfig.RequireSignature
		config.AppConfig.RequireSignature = true
		defer func() { config.AppConfig.RequireSignature = orig }()

		path := "/secure.png"
		exp := "1"
		data := fmt.Sprintf("%s%s", path, exp)
		h := hmac.New(sha256.New, []byte(config.AppConfig.SignatureKey))
		h.Write([]byte(data))
		sig := hex.EncodeToString(h.Sum(nil))

		url := fmt.Sprintf("/uploads%s?sig=%s&exp=%s", path, sig, exp)
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "URL expired")
	})
}
