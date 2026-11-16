package instance_service

import (
	"errors"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_repository "github.com/evolution-foundation/evolution-go/pkg/instance/repository"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/passkey/ceremony"
	whatsmeow_service "github.com/evolution-foundation/evolution-go/pkg/whatsmeow/service"
	"go.mau.fi/whatsmeow"
)

type testQRClient struct {
	loggedIn  bool
	connected bool
}

func (c testQRClient) IsLoggedIn() bool  { return c.loggedIn }
func (c testQRClient) IsConnected() bool { return c.connected }

type testQRRepository struct {
	instance_repository.InstanceRepository
	instance *instance_model.Instance
}

func (r testQRRepository) GetInstanceByID(id string) (*instance_model.Instance, error) {
	return r.instance, nil
}

type testQRWhatsmeowService struct {
	whatsmeow_service.WhatsmeowService
	startCalls int
	startErr   error
	onStart    func(string)
	ceremony   *ceremony.Store
}

func (s *testQRWhatsmeowService) StartInstance(id string) error {
	s.startCalls++
	if s.onStart != nil {
		s.onStart(id)
	}
	return s.startErr
}

func (s *testQRWhatsmeowService) PasskeyCeremonyStore() *ceremony.Store {
	return s.ceremony
}

func newQRTestService(t *testing.T, instance *instance_model.Instance) (instances, *testQRWhatsmeowService) {
	t.Helper()
	whatsmeowService := &testQRWhatsmeowService{}
	return instances{
		instanceRepository: testQRRepository{instance: instance},
		whatsmeowService:   whatsmeowService,
		loggerWrapper:      logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: t.TempDir()}),
		clientPointer:      make(map[string]*whatsmeow.Client),
	}, whatsmeowService
}

func TestGetQrDoesNotRestartLoggedInClient(t *testing.T) {
	for _, connected := range []bool{false, true} {
		t.Run(map[bool]string{false: "disconnected", true: "connected"}[connected], func(t *testing.T) {
			instance := &instance_model.Instance{Id: "test"}
			service, whatsmeowService := newQRTestService(t, instance)

			result, err := service.getQr(instance, testQRClient{loggedIn: true, connected: connected})
			if err == nil || err.Error() != "session already logged in" || result != nil {
				t.Fatalf("got result %v and error %v, want session already logged in", result, err)
			}
			if whatsmeowService.startCalls != 0 {
				t.Fatalf("started client %d times for a logged-in session", whatsmeowService.startCalls)
			}
		})
	}
}

func TestGetQrStartsMissingClientAndReturnsStartError(t *testing.T) {
	instance := &instance_model.Instance{Id: "test"}
	service, whatsmeowService := newQRTestService(t, instance)
	whatsmeowService.startErr = errors.New("start failed")

	result, err := service.GetQr(instance)
	if !errors.Is(err, whatsmeowService.startErr) || result != nil {
		t.Fatalf("got result %v and error %v, want start error", result, err)
	}
	if whatsmeowService.startCalls != 1 {
		t.Fatalf("started client %d times, want 1", whatsmeowService.startCalls)
	}
}

func TestGetQrStartsMissingClientAndReturnsQRCode(t *testing.T) {
	instance := &instance_model.Instance{Id: "test", Qrcode: "image|code"}
	service, whatsmeowService := newQRTestService(t, instance)
	whatsmeowService.onStart = func(id string) {
		service.clientPointer[id] = &whatsmeow.Client{}
	}

	result, err := service.GetQr(instance)
	if err != nil {
		t.Fatal(err)
	}
	if result.Qrcode != "image" || result.Code != "code" {
		t.Fatalf("unexpected QR result: %+v", result)
	}
	if whatsmeowService.startCalls != 1 {
		t.Fatalf("started client %d times, want 1", whatsmeowService.startCalls)
	}
}

func TestGetQrReturnsExistingQRCodeWithoutRestart(t *testing.T) {
	instance := &instance_model.Instance{Id: "test", Qrcode: "image|code"}
	service, whatsmeowService := newQRTestService(t, instance)
	service.clientPointer[instance.Id] = &whatsmeow.Client{}

	result, err := service.GetQr(instance)
	if err != nil {
		t.Fatal(err)
	}
	if result.Qrcode != "image" || result.Code != "code" {
		t.Fatalf("unexpected QR result: %+v", result)
	}
	if whatsmeowService.startCalls != 0 {
		t.Fatalf("started existing client %d times", whatsmeowService.startCalls)
	}
}

func TestGetQrReturnsActivePasskeyWithoutRestart(t *testing.T) {
	instance := &instance_model.Instance{Id: "test"}
	service, whatsmeowService := newQRTestService(t, instance)
	service.clientPointer[instance.Id] = &whatsmeow.Client{}
	whatsmeowService.ceremony = ceremony.NewStore()
	whatsmeowService.ceremony.Start(instance.Id, nil)
	whatsmeowService.ceremony.SetConfirmation(instance.Id, "123456", false)

	result, err := service.GetQr(instance)
	if err != nil {
		t.Fatal(err)
	}
	if result.PasskeyStage != ceremony.StageConfirmation || result.PasskeyCode != "123456" ||
		!strings.HasPrefix(result.PasskeyOpenURL, "https://web.whatsapp.com/#wapk=") {
		t.Fatalf("unexpected passkey result: %+v", result)
	}
	if whatsmeowService.startCalls != 0 {
		t.Fatalf("started existing client %d times", whatsmeowService.startCalls)
	}
}
