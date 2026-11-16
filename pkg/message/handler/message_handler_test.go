package message_handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	message_service "github.com/evolution-foundation/evolution-go/pkg/message/service"
	"github.com/gin-gonic/gin"
)

type countService struct {
	message_service.MessageService
	instanceID string
	err        error
}

func (s *countService) CountTrackedMessages(instance *instance_model.Instance) (int64, error) {
	s.instanceID = instance.Id
	return 3, s.err
}

func TestCountTrackedMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name       string
		enabled    bool
		err        error
		wantStatus int
		wantBody   string
	}{
		{"enabled", true, nil, http.StatusOK, `"data":3`},
		{"disabled", false, nil, http.StatusServiceUnavailable, "message tracking is disabled"},
		{"database error", true, errors.New("database failure"), http.StatusInternalServerError, "database failure"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &countService{err: test.err}
			handler := NewMessageHandler(service, &config.Config{DatabaseSaveMessages: test.enabled})
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Set("instance", &instance_model.Instance{Id: "instance-1"})
			handler.CountTrackedMessages(ctx)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("got HTTP %d: %s", response.Code, response.Body.String())
			}
			if test.enabled && service.instanceID != "instance-1" {
				t.Fatalf("count used instance %q", service.instanceID)
			}
		})
	}
}
