package core

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lazyautoops/lazy-auto-ops/internal/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db     *gorm.DB
	config config.JWTConfig
}

type Claims struct {
	UserID              string `json:"user_id"`
	Username            string `json:"username"`
	RoleCode            string `json:"role_code"`
	ForcePasswordChange bool   `json:"force_password_change"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token                   string `json:"token"`
	Expire                  int64  `json:"expire"`
	UserInfo                *User  `json:"user_info"`
	RecommendChangePassword bool   `json:"recommend_change_password"`
	MustChangePassword      bool   `json:"must_change_password"`
}

func NewAuthService(db *gorm.DB, cfg config.JWTConfig) *AuthService {
	return &AuthService{db: db, config: cfg}
}

// InitDefaultAdmin 初始化默认管理员
func (s *AuthService) InitDefaultAdmin() error {
	// 检查是否已有管理员角色
	var adminRole Role
	if err := s.db.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			adminRole = Role{
				Name:        "管理员",
				Code:        "admin",
				Description: "系统管理员，拥有所有权限",
			}
			if err := s.db.Create(&adminRole).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// 检查是否已有管理员用户
	var count int64
	s.db.Model(&User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	// 创建默认管理员
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := User{
		Username: "admin",
		Password: string(hashedPwd),
		Nickname: "管理员",
		Status:   1,
		RoleID:   adminRole.ID,
	}
	return s.db.Create(&admin).Error
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	var user User
	if err := s.db.Preload("Role").Preload("Role.Permissions").Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	if user.Status != 1 {
		return nil, errors.New("用户已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("密码错误")
	}

	mustChangePassword := isDefaultAdminPassword(&user)

	// 生成Token
	expire := time.Now().Add(time.Duration(s.config.Expire) * time.Hour)
	roleCode := ""
	if user.Role != nil {
		roleCode = user.Role.Code
	}

	claims := Claims{
		UserID:              user.ID,
		Username:            user.Username,
		RoleCode:            roleCode,
		ForcePasswordChange: mustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expire),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:                   tokenStr,
		Expire:                  expire.Unix(),
		UserInfo:                &user,
		RecommendChangePassword: mustChangePassword,
		MustChangePassword:      mustChangePassword,
	}, nil
}

// ValidateToken 验证Token
func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.config.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("无效的Token")
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(id string) (*User, error) {
	var user User
	if err := s.db.Preload("Role").Preload("Role.Permissions").First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// NeedPasswordChange 当前用户是否仍在使用默认密码
func (s *AuthService) NeedPasswordChange(id string) (bool, error) {
	var user User
	if err := s.db.Select("id", "username", "password").First(&user, "id = ?", id).Error; err != nil {
		return false, err
	}
	return isDefaultAdminPassword(&user), nil
}

func isDefaultAdminPassword(user *User) bool {
	if user == nil || !strings.EqualFold(user.Username, "admin") {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("admin123")) == nil
}
