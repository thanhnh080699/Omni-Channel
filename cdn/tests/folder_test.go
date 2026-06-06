package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderManagement(t *testing.T) {
	router := SetupTestRouter()
	SetupTestDirs()
	defer CleanupTestDirs()

	t.Run("Create Folder (Unauthenticated)", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"path": "test-folder"})
		req, _ := http.NewRequest("POST", "/api/folder", bytes.NewBuffer(body))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Create Folder (Success)", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"path": "test-folder"})
		req, _ := http.NewRequest("POST", "/api/folder", bytes.NewBuffer(body))
		req.Header.Set("X-API-KEY", "test_key")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		assert.DirExists(t, "./test_uploads/test-folder")
	})

	t.Run("Rename Folder (Success)", func(t *testing.T) {
		// Mock folder creation first
		os.MkdirAll("./test_uploads/old-folder", 0755)

		body, _ := json.Marshal(map[string]string{
			"old_path": "old-folder",
			"new_path": "new-folder",
		})
		req, _ := http.NewRequest("PUT", "/api/folder", bytes.NewBuffer(body))
		req.Header.Set("X-API-KEY", "test_key")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		assert.DirExists(t, "./test_uploads/new-folder")
		assert.NoDirExists(t, "./test_uploads/old-folder")
	})

	t.Run("Delete Folder (Success)", func(t *testing.T) {
		// Mock folder creation
		os.MkdirAll("./test_uploads/delete-me", 0755)

		body, _ := json.Marshal(map[string]string{"path": "delete-me"})
		req, _ := http.NewRequest("DELETE", "/api/folder", bytes.NewBuffer(body))
		req.Header.Set("X-API-KEY", "test_key")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		assert.NoDirExists(t, "./test_uploads/delete-me")
	})
}
