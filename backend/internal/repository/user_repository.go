package repository

import (
	"context"
	"errors"
	"fmt"

	"fiber-otdr-fault-localization/backend/internal/config"
	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrNotFound = errors.New("record not found")

// ErrConcurrentChange means a version-guarded conditional update matched no
// row because another transaction committed first.
var ErrConcurrentChange = errors.New("record changed concurrently")

type Store struct {
	DB     *gorm.DB
	Users  *UserRepository
	Routes *FiberRouteRepository
	Traces *TraceRepository
	Events *EventRepository
	Cases  *CaseRepository
	Audits *AuditRepository
}

func Open(cfg config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBDriver == "sqlite" {
		dialector = sqlite.Open(cfg.DBDSN)
	} else {
		dialector = postgres.Open(cfg.DBDSN)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Warn), TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("obtain sql database: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	return db, nil
}

func NewStore(db *gorm.DB) *Store {
	return &Store{DB: db, Users: &UserRepository{db}, Routes: &FiberRouteRepository{db}, Traces: &TraceRepository{db}, Events: &EventRepository{db}, Cases: &CaseRepository{db}, Audits: &AuditRepository{db}}
}

func (s *Store) Transaction(fn func(*Store) error) error {
	return s.DB.Transaction(func(tx *gorm.DB) error { return fn(NewStore(tx)) })
}

func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return fmt.Errorf("obtain sql database for readiness: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database readiness ping: %w", err)
	}
	return nil
}

func MigrateAndSeed(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.User{}, &model.FiberRoute{}, &model.TraceCapture{}, &model.EventMarker{}, &model.LocalizationCase{}, &model.AuditLog{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	accounts := []struct{ username, display, role string }{{"analyst", "分析员", constants.RoleAnalyst}, {"reviewer", "复核员", constants.RoleReviewer}, {"admin", "系统管理员", constants.RoleAdmin}}
	hash, err := bcrypt.GenerateFromPassword([]byte("DemoPass123!"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}
	for _, account := range accounts {
		var existing model.User
		err := db.Where("username = ?", account.username).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find seed user: %w", err)
		}
		user := model.User{Username: account.username, DisplayName: account.display, Role: account.role, PasswordHash: string(hash), Active: true}
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("create seed user %s: %w", account.username, err)
		}
	}
	return nil
}

type UserRepository struct{ db *gorm.DB }

func (r *UserRepository) FindByUsername(username string) (model.User, error) {
	var user model.User
	if err := r.db.Where("username = ? AND active = ?", username, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, ErrNotFound
		}
		return user, fmt.Errorf("find user by username: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(id uint) (model.User, error) {
	var user model.User
	if err := r.db.Where("id = ? AND active = ?", id, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, ErrNotFound
		}
		return user, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}
