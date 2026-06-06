package tests

import (
	"encoding/json"
	"image/color"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
)

func TestFileUpload(t *testing.T) {
	router := SetupTestRouter()
	SetupTestDirs()
	defer CleanupTestDirs()

	t.Run("Single Upload (Success)", func(t *testing.T) {
		pr, pw := io.Pipe()
		m := multipart.NewWriter(pw)

		go func() {
			defer pw.Close()
			defer m.Close()

			// Use real image for Deep Inspection
			// Create dummy image data
			part, _ := m.CreateFormFile("file", "test.png")
			img := imaging.New(100, 100, color.White)
			imaging.Encode(part, img, imaging.PNG)
			m.WriteField("folder", "uploads-test")
		}()

		req, _ := http.NewRequest("POST", "/api/upload", pr)
		req.Header.Set("Content-Type", m.FormDataContentType())
		req.Header.Set("X-API-KEY", "test_key")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Response body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var res map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.Nil(t, err)

		data, ok := res["data"].(map[string]interface{})
		assert.True(t, ok, "data field should be a map")
		if ok {
			path, ok := data["path"].(string)
			assert.True(t, ok, "path should be a string")
			assert.Contains(t, path, "uploads-test/")
			assert.FileExists(t, filepath.Join("./test_uploads", path))
		}
	})

	t.Run("Deep Inspection (Fail - Not an Image)", func(t *testing.T) {
		pr, pw := io.Pipe()
		m := multipart.NewWriter(pw)

		go func() {
			defer pw.Close()
			defer m.Close()

			// Non-image content
			part, _ := m.CreateFormFile("file", "test.jpg")
			part.Write([]byte("this is a plain text file, not a jpg"))
		}()

		req, _ := http.NewRequest("POST", "/api/upload", pr)
		req.Header.Set("Content-Type", m.FormDataContentType())
		req.Header.Set("X-API-KEY", "test_key")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "not a valid image")
	})
}
