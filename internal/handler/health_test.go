package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func TestPing_DBNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := NewHealth(nil) // db == nil
	h.SetupRoutes(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestPing_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDBPinger(ctrl)
	db.EXPECT().Ping(gomock.Any()).Return(nil)

	h := NewHealth(db)
	h.SetupRoutes(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestPing_Fail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDBPinger(ctrl)
	db.EXPECT().Ping(gomock.Any()).Return(errors.New("db down"))

	h := NewHealth(db)
	h.SetupRoutes(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
