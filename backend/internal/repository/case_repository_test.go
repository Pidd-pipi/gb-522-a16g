package repository

import (
	"testing"
	"time"

	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecoverStaleAnalysisHonorsLease(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:recover-lease?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.LocalizationCase{}); err != nil {
		t.Fatal(err)
	}
	old := model.LocalizationCase{RouteID: 1, BaselineTraceID: 1, CurrentTraceID: 2, CaseStatus: constants.CaseAnalyzing, ParametersJSON: []byte(`{}`), Version: 2, CreatedBy: 1}
	fresh := model.LocalizationCase{RouteID: 1, BaselineTraceID: 1, CurrentTraceID: 2, CaseStatus: constants.CaseAnalyzing, ParametersJSON: []byte(`{}`), Version: 2, CreatedBy: 1}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&fresh).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&old).Update("updated_at", time.Now().Add(-20*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	repo := &CaseRepository{db: db}
	recovered, err := repo.RecoverStaleAnalysis(old.ID, time.Now().Add(-10*time.Minute))
	if err != nil || !recovered {
		t.Fatalf("expected old lease recovery, recovered=%v err=%v", recovered, err)
	}
	recovered, err = repo.RecoverStaleAnalysis(fresh.ID, time.Now().Add(-10*time.Minute))
	if err != nil || recovered {
		t.Fatalf("fresh lease must stay active, recovered=%v err=%v", recovered, err)
	}
	var reloaded model.LocalizationCase
	if err := db.First(&reloaded, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.CaseStatus != constants.CaseDraft || reloaded.Version != 3 {
		t.Fatalf("unexpected recovered case: status=%s version=%d", reloaded.CaseStatus, reloaded.Version)
	}
}
