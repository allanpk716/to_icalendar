package microsofttodo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenStatus Token 状态信息
type TokenStatus struct {
	HasToken      bool          `json:"has_token"`
	IsExpired     bool          `json:"is_expired"`
	NeedsRefresh  bool          `json:"needs_refresh"`
	NeedsReauth   bool          `json:"needs_reauth"`
	ExpiresAt     time.Time     `json:"expires_at"`
	TimeToExpiry  time.Duration `json:"time_to_expiry"`
	LastRefresh   time.Time     `json:"last_refresh"`
	RefreshCount  int           `json:"refresh_count"`
}

// TokenManagerConfig Token 管理器配置
type TokenManagerConfig struct {
	// 提前多少小时开始刷新 token
	RefreshBeforeExpiry int `yaml:"refresh_before_expiry" default:"24"`
	// 检查间隔，单位小时
	CheckInterval int `yaml:"check_interval" default:"6"`
	// 最大重试次数
	MaxRetries int `yaml:"max_retries" default:"3"`
	// 重试间隔，单位分钟
	RetryInterval int `yaml:"retry_interval" default:"30"`
	// 是否启用主动刷新
	Enabled bool `yaml:"enabled" default:"true"`
}

// TokenManager 统一管理 token 的生命周期
type TokenManager struct {
	client    *SimpleTodoClient
	refresher *TokenRefresher
	config    *TokenManagerConfig
	logger    Logger

	// 事件回调
	onTokenRefreshed func(token *TokenData)
	onRefreshFailed  func(error)
	onReauthNeeded   func(error)

	// 状态跟踪
	mutex         sync.RWMutex
	status        *TokenStatus
	refreshCount  int
	lastRefresh   time.Time
}

// NewTokenManager 创建 Token 管理器
func NewTokenManager(client *SimpleTodoClient, config *TokenManagerConfig, logger Logger) *TokenManager {
	// 设置默认配置
	if config == nil {
		config = &TokenManagerConfig{
			Enabled:              true,
			RefreshBeforeExpiry:  24,
			CheckInterval:        6,
			MaxRetries:           3,
			RetryInterval:        30,
		}
	}

	tm := &TokenManager{
		client: client,
		config: config,
		logger: logger,
		status: &TokenStatus{
			HasToken:     false,
			IsExpired:    true,
			NeedsRefresh: false,
			NeedsReauth:  true,
		},
	}

	// 创建刷新器
	tm.refresher = NewTokenRefresher(client, config, logger)

	// 设置刷新器的回调
	tm.refresher.OnTokenRefreshed(func(token *TokenData) {
		tm.handleTokenRefreshed(token)
	})
	tm.refresher.OnRefreshFailed(func(err error) {
		tm.handleRefreshFailed(err)
	})
	tm.refresher.OnReauthNeeded(func(err error) {
		tm.handleReauthNeeded(err)
	})

	return tm
}

// Start 启动 Token 管理器
func (tm *TokenManager) Start(ctx context.Context) error {
	if !tm.config.Enabled {
		tm.logger.Infof("Token 主动刷新已禁用")
		return nil
	}

	// 立即更新一次状态
	tm.updateStatus(ctx)

	// 启动后台刷新器
	if err := tm.refresher.Start(ctx); err != nil {
		return fmt.Errorf("启动 token 刷新器失败: %w", err)
	}

	tm.logger.Infof("Token 管理器已启动")
	return nil
}

// Stop 停止 Token 管理器
func (tm *TokenManager) Stop() error {
	if err := tm.refresher.Stop(); err != nil {
		return fmt.Errorf("停止 token 刷新器失败: %w", err)
	}

	tm.logger.Infof("Token 管理器已停止")
	return nil
}

// GetStatus 获取当前 Token 状态
func (tm *TokenManager) GetStatus() *TokenStatus {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	// 返回状态的副本
	status := *tm.status
	return &status
}

// RefreshTokenNow 立即刷新 Token
func (tm *TokenManager) RefreshTokenNow(ctx context.Context) error {
	return tm.refresher.CheckAndRefreshNow(ctx)
}

// updateStatus 更新 Token 状态
func (tm *TokenManager) updateStatus(ctx context.Context) {
	tokenStatus, err := tm.client.GetTokenStatus()
	if err != nil {
		tm.logger.Debugf("获取 token 状态失败: %v", err)
		tm.mutex.Lock()
		tm.status = &TokenStatus{
			HasToken:     false,
			IsExpired:    true,
			NeedsRefresh: false,
			NeedsReauth:  true,
		}
		tm.mutex.Unlock()
		return
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 更新状态
	tm.status = tokenStatus
	tm.status.LastRefresh = tm.lastRefresh
	tm.status.RefreshCount = tm.refreshCount
}

// 事件回调设置方法
func (tm *TokenManager) OnTokenRefreshed(callback func(token *TokenData)) {
	tm.onTokenRefreshed = callback
}

func (tm *TokenManager) OnRefreshFailed(callback func(error)) {
	tm.onRefreshFailed = callback
}

func (tm *TokenManager) OnReauthNeeded(callback func(error)) {
	tm.onReauthNeeded = callback
}

// 事件处理器
func (tm *TokenManager) handleTokenRefreshed(token *TokenData) {
	tm.mutex.Lock()
	tm.lastRefresh = time.Now()
	tm.refreshCount++
	tm.mutex.Unlock()

	tm.logger.Infof("Token 刷新成功")

	// 更新状态
	tm.updateStatus(context.Background())

	// 调用回调
	if tm.onTokenRefreshed != nil {
		tm.onTokenRefreshed(token)
	}
}

func (tm *TokenManager) handleRefreshFailed(err error) {
	tm.logger.Warnf("Token 刷新失败: %v", err)

	// 调用回调
	if tm.onRefreshFailed != nil {
		tm.onRefreshFailed(err)
	}
}

func (tm *TokenManager) handleReauthNeeded(err error) {
	tm.logger.Errorf("需要重新认证: %v", err)

	// 更新状态
	tm.mutex.Lock()
	tm.status.NeedsReauth = true
	tm.mutex.Unlock()

	// 调用回调
	if tm.onReauthNeeded != nil {
		tm.onReauthNeeded(err)
	}
}

// GetConfig 获取配置
func (tm *TokenManager) GetConfig() *TokenManagerConfig {
	return tm.config
}

// IsEnabled 检查是否启用
func (tm *TokenManager) IsEnabled() bool {
	return tm.config.Enabled
}

// Logger 接口定义，避免循环依赖
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// DefaultTokenManagerConfig 返回默认配置
func DefaultTokenManagerConfig() *TokenManagerConfig {
	return &TokenManagerConfig{
		Enabled:              true,
		RefreshBeforeExpiry:  24,
		CheckInterval:        6,
		MaxRetries:           3,
		RetryInterval:        30,
	}
}

// ValidateTokenManagerConfig 验证配置
func ValidateTokenManagerConfig(config *TokenManagerConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	if config.RefreshBeforeExpiry <= 0 {
		return fmt.Errorf("刷新提前时间必须大于0")
	}

	if config.CheckInterval <= 0 {
		return fmt.Errorf("检查间隔必须大于0")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("最大重试次数不能小于0")
	}

	if config.RetryInterval <= 0 {
		return fmt.Errorf("重试间隔必须大于0")
	}

	return nil
}