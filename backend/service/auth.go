package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"

	"finance-system/database"
	"finance-system/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

// SetJWTSecret 设置JWT密钥（必须调用）
func SetJWTSecret(secret string) {
	if secret == "" {
		panic("JWT密钥不能为空，请设置 JWT_SECRET 环境变量")
	}
	if len(secret) < 32 {
		panic("JWT密钥长度至少32位")
	}
	jwtSecret = []byte(secret)
}

// AuthService 认证服务
type AuthService struct{}

// NewAuthService 创建认证服务
func NewAuthService() *AuthService {
	return &AuthService{}
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 20 {
		return errors.New("用户名长度必须在3-20位之间")
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	if !matched {
		return errors.New("用户名只能包含字母、数字和下划线")
	}
	return nil
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("密码长度至少6位")
	}
	if len(password) > 50 {
		return errors.New("密码长度不能超过50位")
	}
	return nil
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	if len(email) > 100 {
		return errors.New("邮箱长度不能超过100位")
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email)
	if !matched {
		return errors.New("邮箱格式不正确")
	}
	return nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, username, password, email string) (*model.User, error) {
	// 验证输入
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	// 检查用户名是否存在
	var existingID int64
	err := database.DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&existingID)
	if err == nil {
		return nil, errors.New("用户名已存在")
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 检查邮箱是否存在
	err = database.DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&existingID)
	if err == nil {
		return nil, errors.New("邮箱已被注册")
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result, err := database.DB.Exec(
		"INSERT INTO users (username, password, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		username, string(hashedPassword), email, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:        id,
		Username:  username,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *model.User, error) {
	var user model.User
	var hashedPassword string

	err := database.DB.QueryRow(
		"SELECT id, username, password, email, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &hashedPassword, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return "", nil, errors.New("用户名或密码错误")
	}
	if err != nil {
		return "", nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	// 生成 JWT Token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", nil, err
	}

	return token, &user, nil
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	err := database.DB.QueryRow(
		"SELECT id, username, email, created_at, updated_at FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("用户不存在")
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// generateToken 生成JWT Token
func (s *AuthService) generateToken(userID int64) (string, error) {
	if jwtSecret == nil {
		return "", errors.New("JWT密钥未配置")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken 验证JWT Token
func (s *AuthService) ValidateToken(tokenString string) (int64, error) {
	if jwtSecret == nil {
		return 0, errors.New("JWT密钥未配置")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int64(claims["user_id"].(float64))
		return userID, nil
	}

	return 0, errors.New("无效的Token")
}

// EmailExists 检查邮箱是否已存在
func (s *AuthService) EmailExists(email string) bool {
	var existingID int64
	err := database.DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&existingID)
	return err == nil
}
