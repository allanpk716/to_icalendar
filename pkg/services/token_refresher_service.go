package services

import (
	"context"
	"fmt"

	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/microsofttodo"
	"github.com/allanpk716/to_icalendar/pkg/models"
)

// TokenRefresherService Token 刷新服务接口
type TokenRefresherService interface {
	// Start 启动服务
	Start() error
	// Stop 停止服务
	Stop() error
	// GetStatus 获取刷新器状态
	GetStatus() map[string]interface{}
	// RefreshTokenNow 立即刷新 Token
	RefreshTokenNow(ctx context.Context) error
	// IsEnabled 检查是否启用
	IsEnabled() bool
	// SetReauthCallback 设置重新认证回调
	SetReauthCallback(callback func(error))
}

// tokenRefresherServiceImpl Token 刷新服务实现
type tokenRefresherServiceImpl struct {
	tokenManager *microsofttodo.TokenManager
	logger       logger.Logger
	ctx          context.Context
	cancel       context.CancelFunc

	// 回调函数
	reauthCallback func(error)
}

// NewTokenRefresherService 创建 Token 刷新服务
func NewTokenRefresherService(
	config *models.ServerConfig,
	logger logger.Logger,
) TokenRefresherService {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建 Token 管理器配置
	var tokenManagerConfig *microsofttodo.TokenManagerConfig
	if config.TokenManager != nil {
		// 从 models.TokenManagerConfig 转换为 microsofttodo.TokenManagerConfig
		tokenManagerConfig = &microsofttodo.TokenManagerConfig{
			Enabled:              config.TokenManager.Enabled,
			RefreshBeforeExpiry:  config.TokenManager.RefreshBeforeExpiry,
			CheckInterval:        config.TokenManager.CheckInterval,
			MaxRetries:           config.TokenManager.MaxRetries,
			RetryInterval:        config.TokenManager.RetryInterval,
		}
	} else {
		// 使用默认配置
		tokenManagerConfig = microsofttodo.DefaultTokenManagerConfig()
	}

	// 创建一个临时的 Todo 客户端用于 token 管理
	// 注意：这里我们需要访问 token 管理功能，但不一定要创建完整的 Todo 服务
	// 我们将使用 SimpleTodoClient 来处理 token
	todoClient, err := microsofttodo.NewSimpleTodoClient(
		config.MicrosoftTodo.TenantID,
		config.MicrosoftTodo.ClientID,
		config.MicrosoftTodo.ClientSecret,
		config.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		logger.Errorf("创建 Todo 客户端失败: %v", err)
		return &tokenRefresherServiceImpl{
			logger: logger,
			ctx:    ctx,
			cancel: cancel,
		}
	}

	// 创建 Token 管理器
	tokenManager := microsofttodo.NewTokenManager(todoClient, tokenManagerConfig, logger)

	// 设置事件回调
	tokenManager.OnRefreshFailed(func(err error) {
		logger.Warnf("Token 刷新失败: %v", err)
	})

	return &tokenRefresherServiceImpl{
		tokenManager: tokenManager,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start 启动服务
func (s *tokenRefresherServiceImpl) Start() error {
	if s.tokenManager == nil {
		return fmt.Errorf("Token 管理器未初始化")
	}

	if err := s.tokenManager.Start(s.ctx); err != nil {
		return fmt.Errorf("启动 Token 管理器失败: %w", err)
	}

	s.logger.Info("Token 刷新服务已启动")
	return nil
}

// Stop 停止服务
func (s *tokenRefresherServiceImpl) Stop() error {
	if s.tokenManager == nil {
		return nil
	}

	s.cancel()

	if err := s.tokenManager.Stop(); err != nil {
		return fmt.Errorf("停止 Token 管理器失败: %w", err)
	}

	s.logger.Info("Token 刷新服务已停止")
	return nil
}

// GetStatus 获取刷新器状态
func (s *tokenRefresherServiceImpl) GetStatus() map[string]interface{} {
	if s.tokenManager == nil {
		return map[string]interface{}{
			"initialized": false,
			"error":       "Token 管理器未初始化",
		}
	}

	status := s.tokenManager.GetStatus()
	config := s.tokenManager.GetConfig()

	return map[string]interface{}{
		"initialized": true,
		"enabled":     config.Enabled,
		"token":       status,
		"config": map[string]interface{}{
			"refresh_before_expiry": config.RefreshBeforeExpiry,
			"check_interval":        config.CheckInterval,
			"max_retries":           config.MaxRetries,
			"retry_interval":        config.RetryInterval,
		},
	}
}

// RefreshTokenNow 立即刷新 Token
func (s *tokenRefresherServiceImpl) RefreshTokenNow(ctx context.Context) error {
	if s.tokenManager == nil {
		return fmt.Errorf("Token 管理器未初始化")
	}

	return s.tokenManager.RefreshTokenNow(ctx)
}

// IsEnabled 检查是否启用
func (s *tokenRefresherServiceImpl) IsEnabled() bool {
	if s.tokenManager == nil {
		return false
	}

	return s.tokenManager.IsEnabled()
}

// SetReauthCallback 设置重新认证回调
func (s *tokenRefresherServiceImpl) SetReauthCallback(callback func(error)) {
	s.reauthCallback = callback

	if s.tokenManager != nil {
		s.tokenManager.OnReauthNeeded(func(err error) {
			s.logger.Warnf("需要重新认证: %v", err)
			if s.reauthCallback != nil {
				s.reauthCallback(err)
			}
		})
	}
}