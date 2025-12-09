package microsofttodo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenRefresher 后台 Token 刷新器
type TokenRefresher struct {
	client    *SimpleTodoClient
	config    *TokenManagerConfig
	logger    Logger

	// 控制定时器
	ticker    *time.Ticker
	stopCh    chan struct{}
	running   bool
	mutex     sync.RWMutex

	// 刷新状态
	lastCheck time.Time
	nextCheck time.Time

	// 事件回调
	onTokenRefreshed func(token *TokenData)
	onRefreshFailed  func(error)
	onReauthNeeded   func(error)

	// 防止并发刷新
	refreshMutex sync.Mutex
}

// NewTokenRefresher 创建 Token 刷新器
func NewTokenRefresher(client *SimpleTodoClient, config *TokenManagerConfig, logger Logger) *TokenRefresher {
	return &TokenRefresher{
		client: client,
		config: config,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start 启动后台刷新任务
func (tr *TokenRefresher) Start(ctx context.Context) error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	if tr.running {
		return nil
	}

	tr.running = true
	tr.logger.Debugf("启动 Token 刷新器，检查间隔: %d 小时", tr.config.CheckInterval)

	// 立即执行一次检查
	go tr.checkAndRefresh(ctx)

	// 启动定时检查
	interval := time.Duration(tr.config.CheckInterval) * time.Hour
	tr.ticker = time.NewTicker(interval)
	tr.nextCheck = time.Now().Add(interval)

	go tr.run(ctx)

	return nil
}

// Stop 停止后台刷新任务
func (tr *TokenRefresher) Stop() error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	if !tr.running {
		return nil
	}

	tr.running = false
	close(tr.stopCh)

	if tr.ticker != nil {
		tr.ticker.Stop()
	}

	tr.logger.Debugf("Token 刷新器已停止")
	return nil
}

// CheckAndRefreshNow 立即检查并刷新（如果需要）
func (tr *TokenRefresher) CheckAndRefreshNow(ctx context.Context) error {
	return tr.checkAndRefresh(ctx)
}

// run 运行定时检查循环
func (tr *TokenRefresher) run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			tr.logger.Errorf("Token 刷新器运行时出现异常: %v", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			tr.logger.Infof("Token 刷新器收到退出信号")
			return
		case <-tr.stopCh:
			tr.logger.Infof("Token 刷新器收到停止信号")
			return
		case <-tr.ticker.C:
			// 定时检查
			tr.logger.Debugf("执行定时 Token 检查")
			if err := tr.checkAndRefresh(ctx); err != nil {
				tr.logger.Errorf("定时 Token 检查失败: %v", err)
			}
		}
	}
}

// checkAndRefresh 检查并刷新 Token
func (tr *TokenRefresher) checkAndRefresh(ctx context.Context) error {
	// 防止并发刷新
	tr.refreshMutex.Lock()
	defer tr.refreshMutex.Unlock()

	defer func() {
		tr.lastCheck = time.Now()
		interval := time.Duration(tr.config.CheckInterval) * time.Hour
		tr.nextCheck = tr.lastCheck.Add(interval)
	}()

	// 加载当前 Token
	token, err := tr.client.loadCachedToken()
	if err != nil {
		tr.logger.Debugf("无法加载缓存的 Token: %v", err)
		return nil // 不是错误，可能还没有 Token
	}

	if token == nil {
		tr.logger.Debugf("没有缓存的 Token")
		return nil
	}

	// 检查是否需要刷新
	needsRefresh, refreshReason := tr.shouldRefreshToken(token)
	if !needsRefresh {
		tr.logger.Debugf("Token 不需要刷新，原因: %s", refreshReason)
		return nil
	}

	tr.logger.Infof("Token 需要刷新，原因: %s", refreshReason)
	if err := tr.refreshTokenWithRetry(ctx, token); err != nil {
		tr.logger.Errorf("Token 刷新失败: %v", err)
		tr.handleRefreshFailure(err)
		return err
	}

	tr.logger.Debugf("Token 刷新成功")
	tr.handleTokenRefreshed(token)
	return nil
}

// shouldRefreshToken 判断是否需要刷新 Token
func (tr *TokenRefresher) shouldRefreshToken(token *TokenData) (bool, string) {
	// 检查 Token 是否已过期
	if token.isExpired() {
		return true, "Token 已过期"
	}

	// 检查是否在指定的刷新时间窗口内
	refreshTime := token.ExpiresAt.Add(-time.Duration(tr.config.RefreshBeforeExpiry) * time.Hour)
	if time.Now().After(refreshTime) {
		return true, fmt.Sprintf("Token 将在 %d 小时内过期", tr.config.RefreshBeforeExpiry)
	}

	// 检查 Refresh Token 是否为空
	if token.RefreshToken == "" {
		return true, "Refresh Token 为空"
	}

	return false, "Token 有效且未到刷新时间"
}

// refreshTokenWithRetry 带重试的 Token 刷新
func (tr *TokenRefresher) refreshTokenWithRetry(ctx context.Context, token *TokenData) error {
	var lastErr error

	for attempt := 0; attempt < tr.config.MaxRetries; attempt++ {
		if attempt > 0 {
			tr.logger.Infof("重试 Token 刷新 (%d/%d)", attempt+1, tr.config.MaxRetries)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(tr.config.RetryInterval) * time.Minute):
				// 继续重试
			}
		}

		// 尝试刷新 Token
		newAccessToken, err := tr.client.refreshAccessToken(ctx, token.RefreshToken)
		if err == nil {
			// 刷新成功，更新 Token
			token.AccessToken = newAccessToken
			// Token 默认有效期为 1 小时
			token.ExpiresAt = time.Now().Add(1 * time.Hour)

			// 保存更新后的 Token
			// expiresIn 参数：1小时 = 3600秒
			if err := tr.client.saveCachedToken(token.AccessToken, token.RefreshToken, 3600); err != nil {
				tr.logger.Errorf("保存刷新后的 Token 失败: %v", err)
				return fmt.Errorf("保存 Token 失败: %w", err)
			}

			return nil
		}

		lastErr = err
		tr.logger.Debugf("Token 刷新尝试 %d 失败: %v", attempt+1, err)

		// 检查是否是认证错误，如果是，不需要重试
		if isAuthError(err) {
			tr.logger.Errorf("Token 刷新失败，需要重新认证: %v", err)
			tr.handleReauthNeeded(err)
			return err
		}
	}

	return fmt.Errorf("Token 刷新失败，已重试 %d 次: %w", tr.config.MaxRetries, lastErr)
}


// isAuthError 检查是否是认证错误
func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	authErrors := []string{
		"invalid_grant",
		"invalid_client",
		"unauthorized_client",
		"authentication_failed",
		"refresh_token_expired",
		"invalid_refresh_token",
	}

	for _, authErr := range authErrors {
		if contains(errStr, authErr) {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    len(s) > len(substr) &&
		    (s[:len(substr)] == substr ||
		     s[len(s)-len(substr):] == substr ||
		     findSubstring(s, substr)))
}

// findSubstring 在字符串中查找子字符串
func findSubstring(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower 转换为小写
func toLower(s string) string {
	result := make([]rune, len([]rune(s)))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + ('a' - 'A')
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// 事件回调设置方法
func (tr *TokenRefresher) OnTokenRefreshed(callback func(token *TokenData)) {
	tr.onTokenRefreshed = callback
}

func (tr *TokenRefresher) OnRefreshFailed(callback func(error)) {
	tr.onRefreshFailed = callback
}

func (tr *TokenRefresher) OnReauthNeeded(callback func(error)) {
	tr.onReauthNeeded = callback
}

// 事件处理器
func (tr *TokenRefresher) handleTokenRefreshed(token *TokenData) {
	if tr.onTokenRefreshed != nil {
		tr.onTokenRefreshed(token)
	}
}

func (tr *TokenRefresher) handleRefreshFailure(err error) {
	if tr.onRefreshFailed != nil {
		tr.onRefreshFailed(err)
	}
}

func (tr *TokenRefresher) handleReauthNeeded(err error) {
	if tr.onReauthNeeded != nil {
		tr.onReauthNeeded(err)
	}
}

// GetStatus 获取刷新器状态
func (tr *TokenRefresher) GetStatus() map[string]interface{} {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()

	return map[string]interface{}{
		"running":    tr.running,
		"lastCheck":  tr.lastCheck,
		"nextCheck":  tr.nextCheck,
		"config": map[string]interface{}{
			"checkInterval":        tr.config.CheckInterval,
			"refreshBeforeExpiry":  tr.config.RefreshBeforeExpiry,
			"maxRetries":           tr.config.MaxRetries,
			"retryInterval":        tr.config.RetryInterval,
			"enabled":              tr.config.Enabled,
		},
	}
}