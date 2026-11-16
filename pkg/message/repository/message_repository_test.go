package message_repository

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	message_model "github.com/evolution-foundation/evolution-go/pkg/message/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestInsertMessagePreservesReferralOnStatusUpdate(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock db: %v", err)
	}

	defer sqlDB.Close()

	var logBuffer bytes.Buffer
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:             sqlDB,
		WithoutReturning: true,
	}), &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
		Logger: gormlogger.New(
			log.New(&logBuffer, "", 0),
			gormlogger.Config{LogLevel: gormlogger.Info, Colorful: false},
		),
	})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}

	repo := NewMessageRepository(gormDB)
	referral := json.RawMessage(`{"ctwaClid":"abc123","showAdAttribution":true}`)

	initial := message_model.Message{
		MessageID: "msg-1",
		Timestamp: "2026-05-09 10:00:00",
		Status:    "Received",
		Source:    "1551999999999",
		Referral:  referral,
	}

	if err := repo.InsertMessage(initial); err != nil {
		t.Fatalf("insert initial message: %v", err)
	}

	initialSQL := logBuffer.String()
	if !strings.Contains(initialSQL, `"referral"="excluded"."referral"`) {
		t.Fatalf("expected initial upsert SQL to update referral, got %q", initialSQL)
	}

	logBuffer.Reset()

	updated := message_model.Message{
		MessageID: "msg-1",
		Timestamp: "2026-05-09 10:05:00",
		Status:    "Read",
		Source:    "1551999999999",
	}

	if err := repo.InsertMessage(updated); err != nil {
		t.Fatalf("insert updated message: %v", err)
	}

	updatedSQL := logBuffer.String()
	if strings.Contains(updatedSQL, `"referral"="excluded"."referral"`) {
		t.Fatalf("expected updated upsert SQL to omit referral update, got %q", updatedSQL)
	}
	if !strings.Contains(updatedSQL, `"timestamp"="excluded"."timestamp","status"="excluded"."status","source"="excluded"."source"`) {
		t.Fatalf("expected updated upsert SQL to keep core columns, got %q", updatedSQL)
	}
}

func TestTrackedMessagesUseInstanceAndIgnoreDuplicates(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, WithoutReturning: true}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMessageRepository(db)
	for _, row := range []struct {
		instance string
		inserted int64
	}{
		{"instance-1", 1},
		{"instance-1", 0},
		{"instance-2", 1},
	} {
		mock.ExpectExec(`INSERT INTO "tracked_messages" .*ON CONFLICT DO NOTHING`).
			WithArgs(row.instance, "message-1").
			WillReturnResult(sqlmock.NewResult(0, row.inserted))
		if err := repo.TrackMessage(row.instance, "message-1"); err != nil {
			t.Fatal(err)
		}
	}
	for _, instance := range []string{"instance-1", "instance-2"} {
		mock.ExpectQuery(`SELECT count\(\*\) FROM "tracked_messages" WHERE instance_id = \$1`).
			WithArgs(instance).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		count, err := repo.CountTrackedMessages(instance)
		if err != nil || count != 1 {
			t.Fatalf("count for %s: got %d, error %v", instance, count, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
