package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/googlecloudplatform/pi-delivery/gen/index"
	"github.com/googlecloudplatform/pi-delivery/pkg/service"
	"go.uber.org/zap"
)

func router(t *testing.T) *gin.Engine {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	t.Cleanup(func() { logger.Sync() })

	service, err := service.NewService(ctx, index.BucketName)
	if err != nil {
		t.Fatalf("NewService() failed: %v", err)
	}
	t.Cleanup(func() { service.Close() })

	return setupRouter(logger, service)
}

func TestGetPi(t *testing.T) {
	router := router(t)
	testCases := []struct {
		name  string
		query string
		want  string
	}{
		{"default", "", "3141592653589793238462643383279502884197169399375105820974944592307816406286208998628034825342117067"},
		{"first 10", "start=0&numberOfDigits=10", "3141592653"},
		{"radix", "start=5&numberOfDigits=10&radix=16", "6a8885a308"},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/v1/pi?%s", tc.query), nil)
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Status is not 200, got = %d", w.Code)
			}
			want := fmt.Sprintf("{\"content\":\"%s\"}", tc.want)
			got := w.Body.String()
			if want != got {
				t.Errorf("Body: want = %s, got = %s", want, got)
			}
		})
	}
}

func TestGetPi_Validation(t *testing.T) {
	router := router(t)
	testCases := []struct {
		name  string
		query string
	}{
		{"negative start", "start=-1"},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/v1/pi?%s", tc.query), nil)
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Status is not 400, got = %d", w.Code)
			}
			t.Logf("body = %s", w.Body.String())
		})
	}
}
