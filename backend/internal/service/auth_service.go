package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"fiber-otdr-fault-localization/backend/internal/config"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	CodeInvalidInput   = "INVALID_INPUT"
	CodeNotFound       = "NOT_FOUND"
	CodeConflict       = "STATE_CONFLICT"
	CodeUnauthorized   = "AUTH_REQUIRED"
	CodeForbidden      = "FORBIDDEN"
	CodeAlgorithmInput = "ALGORITHM_INPUT_INSUFFICIENT"
	// CodeBaselineBlocked is returned when open localization cases still
	// reference the current baseline of a route.
	CodeBaselineBlocked = "BASELINE_CASES_OPEN"
	CodeInternal        = "INTERNAL_ERROR"
)

type AppError struct {
	Code    string
	Status  int
	Message string
	// Details carries structured context such as the blocking cases for a
	// refused baseline switch; omitted from the response payload when nil.
	Details map[string]any
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}
func (e *AppError) Unwrap() error { return e.Err }

func invalid(message string, err error) error {
	return &AppError{CodeInvalidInput, http.StatusBadRequest, message, nil, err}
}
func notFound(resource string) error {
	return &AppError{CodeNotFound, http.StatusNotFound, resource + " does not exist", nil, repository.ErrNotFound}
}
func conflict(message string, err error) error {
	return &AppError{CodeConflict, http.StatusConflict, message, nil, err}
}
func baselineBlocked(message string, blockers []dto.BaselineBlocker) error {
	return &AppError{Code: CodeBaselineBlocked, Status: http.StatusConflict, Message: message, Details: map[string]any{"blockers": blockers}}
}
func internal(message string, err error) error {
	return &AppError{CodeInternal, http.StatusInternalServerError, message, nil, err}
}

type Actor struct {
	ID                        uint
	Username, Role, RequestID string
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	store  *repository.Store
	secret []byte
	ttl    time.Duration
}

func NewAuthService(store *repository.Store, cfg config.Config) *AuthService {
	return &AuthService{store, []byte(cfg.JWTSecret), cfg.AccessTokenTTL}
}

func (s *AuthService) Login(request dto.LoginRequest) (dto.LoginResponse, error) {
	user, err := s.store.Users.FindByUsername(request.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.LoginResponse{}, &AppError{Code: CodeUnauthorized, Status: http.StatusUnauthorized, Message: "username or password is incorrect", Err: err}
		}
		return dto.LoginResponse{}, internal("authentication lookup failed", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return dto.LoginResponse{}, &AppError{Code: CodeUnauthorized, Status: http.StatusUnauthorized, Message: "username or password is incorrect", Err: err}
	}
	now, expires := time.Now(), time.Now().Add(s.ttl)
	claims := Claims{UserID: user.ID, Username: user.Username, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: fmt.Sprint(user.ID), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires), NotBefore: jwt.NewNumericDate(now)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return dto.LoginResponse{}, internal("token signing failed", err)
	}
	return dto.LoginResponse{Token: token, ExpiresAt: expires, User: dto.UserView{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role}}, nil
}

func (s *AuthService) Parse(tokenString string) (Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, &AppError{Code: CodeUnauthorized, Status: http.StatusUnauthorized, Message: "access token is invalid or expired", Err: err}
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return Claims{}, &AppError{Code: CodeUnauthorized, Status: http.StatusUnauthorized, Message: "access token claims are invalid"}
	}
	user, err := s.store.Users.FindByID(claims.UserID)
	if err != nil {
		return Claims{}, &AppError{Code: CodeUnauthorized, Status: http.StatusUnauthorized, Message: "account is inactive", Err: err}
	}
	// Authorization follows the current account record, not role/name claims
	// captured when an older token was issued.
	claims.Username = user.Username
	claims.Role = user.Role
	return *claims, nil
}
