package service

import (
	"errors"
	"go_payment/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret string
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) RegisterUser(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *AuthService) Login(email, password string) (*TokenPair, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	if !user.CheckPassword(password) {
		return nil, errors.New("invalid password")
	}

	// Обновляем время последнего входа
	user.LastLogin = time.Now()
	s.db.Save(&user)

	return s.GenerateTokenPair(&user)
}

func (s *AuthService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	// Создаем Access Token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 1).Unix(),
	})

	accessTokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	// Создаем Refresh Token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	// Сохраняем refresh token в базе
	user.RefreshToken = refreshTokenString
	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

func (s *AuthService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	// Парсим refresh token
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Получаем пользователя
	var user models.User
	if err := s.db.First(&user, claims["user_id"]).Error; err != nil {
		return nil, err
	}

	// Проверяем, совпадает ли refresh token с сохраненным
	if user.RefreshToken != refreshTokenString {
		return nil, errors.New("refresh token has been revoked")
	}

	return s.GenerateTokenPair(&user)
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
}

func (s *AuthService) GetUserPermissions(role models.Role) ([]string, error) {
	var permissions []models.Permission
	err := s.db.
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", role).
		Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	var permissionNames []string
	for _, p := range permissions {
		permissionNames = append(permissionNames, p.Name)
	}
	return permissionNames, nil
}
