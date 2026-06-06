package tests

import (
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
)

// Helper: Create a real dummy image file for testing
func createTestImage(t *testing.T, path string) {
	os.MkdirAll(filepath.Dir(path), 0755)
	img := imaging.New(100, 100, color.White)
	err := imaging.Save(img, path)
	assert.Nil(t, err)
}

func TestImageServingAndProcessing(t *testing.T) {
	router := SetupTestRouter()
	SetupTestDirs()
	defer CleanupTestDirs()

	// Initial image
	createTestImage(t, "./test_uploads/base/test.png")

	t.Run("Serve Static (Success)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/uploads/base/test.png", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Cache-Control"), "max-age")
	})

	t.Run("Serve Processed (MISS -> HIT)", func(t *testing.T) {
		// First time - MISS (renders and caches)
		url := "/uploads/base/test.png?w=200&h=200"
		req, _ := http.NewRequest("GET", url, nil)

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))

		// Check if cache file exists physically
		// relPath: base/test.png
		// cachePath: base/test_w200_h200.png
		cacheFile := "./test_uploads/cache/base/test_w200_h200.png"
		assert.FileExists(t, cacheFile)

		// Second time - HIT (from cache)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
	})

	t.Run("Serve Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/uploads/non-existent.png", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
