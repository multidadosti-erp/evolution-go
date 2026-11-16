package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/evolution-foundation/evolution-go/pkg/config"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRouterWorksWithoutLicenseAndKeepsAdminAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GlobalApiKey: "test-key",
		ClientName:   "test-client",
		LogDirectory: t.TempDir(),
	}
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS poll_votes`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	router := setupRouter(db, sqlDB, nil, cfg, nil, "")

	for _, test := range []struct {
		path string
		key  string
		want int
	}{
		{"/license/status", "", http.StatusNotFound},
		{"/instance/all", "", http.StatusUnauthorized},
		{"/instance/all", "wrong-key", http.StatusUnauthorized},
		{"/instance/all", "test-key", http.StatusOK},
		{"/message/count", "", http.StatusUnauthorized},
	} {
		if test.want == http.StatusOK {
			mock.ExpectQuery(`SELECT .* FROM "instances" WHERE client_name = \$1`).
				WithArgs("test-client").
				WillReturnRows(sqlmock.NewRows([]string{"id"}))
		}
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		req.Header.Set("apikey", test.key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != test.want {
			t.Errorf("%s: got HTTP %d, want %d", test.path, response.Code, test.want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
