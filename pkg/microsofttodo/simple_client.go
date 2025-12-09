package microsofttodo

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/logger"
	timezonepkg "github.com/allanpk716/to_icalendar/pkg/timezone"
)

// AuthConfig 包含 Microsoft Graph API 认证所需的配置
type AuthConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	UserEmail    string
}

// TokenData 存储token信息
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// QueryCacheEntry 查询结果缓存条目
type QueryCacheEntry struct {
	Tasks    []TaskInfo `json:"tasks"`
	CacheTime time.Time `json:"cache_time"`
	TTL      time.Duration `json:"ttl"`
}

// QueryCache 查询结果缓存
type QueryCache struct {
	cache  map[string]*QueryCacheEntry
	mutex  sync.RWMutex
	ttl    time.Duration
}

// NewQueryCache 创建新的查询缓存
func NewQueryCache(ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache: make(map[string]*QueryCacheEntry),
		ttl:   ttl,
	}
}

// Get 获取缓存结果
func (qc *QueryCache) Get(key string) ([]TaskInfo, bool) {
	qc.mutex.RLock()
	defer qc.mutex.RUnlock()

	entry, exists := qc.cache[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Since(entry.CacheTime) > entry.TTL {
		// 过期，删除缓存
		delete(qc.cache, key)
		return nil, false
	}

	return entry.Tasks, true
}

// Set 设置缓存结果
func (qc *QueryCache) Set(key string, tasks []TaskInfo) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.cache[key] = &QueryCacheEntry{
		Tasks:     tasks,
		CacheTime: time.Now(),
		TTL:       qc.ttl,
	}
}

// Clear 清空缓存
func (qc *QueryCache) Clear() {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.cache = make(map[string]*QueryCacheEntry)
}

// GetStats 获取缓存统计信息
func (qc *QueryCache) GetStats() map[string]interface{} {
	qc.mutex.RLock()
	defer qc.mutex.RUnlock()

	totalEntries := len(qc.cache)
	expiredEntries := 0
	now := time.Now()

	for _, entry := range qc.cache {
		if now.Sub(entry.CacheTime) > entry.TTL {
			expiredEntries++
		}
	}

	return map[string]interface{}{
		"total_entries":   totalEntries,
		"expired_entries": expiredEntries,
		"valid_entries":   totalEntries - expiredEntries,
		"ttl_minutes":     qc.ttl.Minutes(),
	}
}

// SimpleTodoClient 简化的 Microsoft Todo 客户端
type SimpleTodoClient struct {
	authConfig  *AuthConfig
	httpClient  *http.Client
	baseURL     string
	queryCache  *QueryCache
}

// NewSimpleTodoClient 创建新的简化 Todo 客户端
func NewSimpleTodoClient(tenantID, clientID, clientSecret, userEmail string) (*SimpleTodoClient, error) {
	// 验证配置
	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("incomplete authentication configuration: tenant_id, client_id, and client_secret are all required")
	}

	return &SimpleTodoClient{
		authConfig: &AuthConfig{
			TenantID:     tenantID,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			UserEmail:    userEmail,
		},
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://graph.microsoft.com/v1.0",
		queryCache: NewQueryCache(5 * time.Minute), // 查询缓存5分钟过期
	}, nil
}

// getAccessToken 获取访问令牌
func (c *SimpleTodoClient) getAccessToken(ctx context.Context) (string, error) {
	// 首先尝试从缓存加载token
	if token, err := c.loadCachedToken(); err == nil && token != nil {
		if !token.isExpired() {
			logger.Debugf("使用缓存的访问令牌")
			return token.AccessToken, nil
		}

		// token过期但refresh token有效，尝试刷新
		if token.RefreshToken != "" {
			logger.Debugf("访问令牌已过期，尝试刷新...")
			if newToken, err := c.refreshAccessToken(ctx, token.RefreshToken); err == nil {
				logger.Debugf("令牌刷新成功")
				return newToken, nil
			}
			logger.Warnf("令牌刷新失败: %v", err)
		}
	}

	// 如果没有有效token或refresh失败，进行交互式认证
	logger.Infof("未找到有效的缓存令牌，开始交互式认证")
	return c.getAccessTokenInteractive(ctx)
}

// getAccessTokenInteractive 使用授权码流程获取访问令牌
func (c *SimpleTodoClient) getAccessTokenInteractive(ctx context.Context) (string, error) {
	fmt.Printf("\n正在启动OAuth2授权码认证流程...\n")

	// 生成PKCE参数
	codeVerifier, codeChallenge := c.generatePKCE()
	authURL := c.buildAuthURLWithPKCE(codeChallenge)

	fmt.Printf("请在浏览器中打开以下URL进行登录授权:\n%s\n\n", authURL)
	fmt.Printf("授权完成后，请复制浏览器地址栏中的完整URL并粘贴到这里:\n")
	fmt.Printf("提示：URL应该包含 'code=' 参数\n\n")

	var authResponse string
	fmt.Scanln(&authResponse)

	// 解析授权响应获取code
	code, err := c.extractAuthCode(authResponse)
	if err != nil {
		return "", fmt.Errorf("failed to extract authorization code: %v", err)
	}

	// 使用code交换access token
	accessToken, err := c.exchangeCodeForTokenWithPKCE(ctx, code, codeVerifier)
	if err != nil {
		return "", err
	}

	// 缓存token以备将来使用
	logger.Infof("认证成功，缓存令牌以备将来使用")
	// 注意：这里简化处理，实际应该保存完整的token信息

	return accessToken, nil
}

// DeviceCodeResponse 设备码响应
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// startDeviceCodeFlow 启动设备码流程
func (c *SimpleTodoClient) startDeviceCodeFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode", c.authConfig.TenantID)

	data := url.Values{}
	data.Set("client_id", c.authConfig.ClientID)
	data.Set("scope", "https://graph.microsoft.com/Tasks.ReadWrite https://graph.microsoft.com/User.Read offline_access")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("device code request failed: %+v", errorResp)
	}

	var deviceCodeResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCodeResp); err != nil {
		return nil, err
	}

	return &deviceCodeResp, nil
}

// waitForDeviceCodeCompletion 等待设备码认证完成
func (c *SimpleTodoClient) waitForDeviceCodeCompletion(ctx context.Context, deviceCode *DeviceCodeResponse) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.authConfig.TenantID)

	interval := time.Duration(deviceCode.Interval) * time.Second
	timeout := 10 * time.Minute // 设置10分钟超时，给用户足够时间

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	for {
		select {
		case <-timeoutCtx.Done():
			return "", fmt.Errorf("device code authentication timed out after %v", time.Since(start))
		case <-time.After(interval):
			// 尝试获取token
			logger.Debugf("检查认证状态... (已用时间: %v)", time.Since(start))
			data := url.Values{}
			data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
			data.Set("client_id", c.authConfig.ClientID)
			data.Set("device_code", deviceCode.DeviceCode)

			req, err := http.NewRequestWithContext(timeoutCtx, "POST", tokenURL, strings.NewReader(data.Encode()))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := c.httpClient.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				// 认证成功，解析token
				logger.Infof("认证成功！ (已用时间: %v)", time.Since(start))
				var tokenResp struct {
					AccessToken string `json:"access_token"`
					TokenType   string `json:"token_type"`
				}

				body, _ := io.ReadAll(resp.Body)
				if err := json.Unmarshal(body, &tokenResp); err == nil && tokenResp.AccessToken != "" {
					return tokenResp.AccessToken, nil
				}
			} else if resp.StatusCode != http.StatusBadRequest {
				resp.Body.Close()
				logger.Warnf("意外的状态码: %d", resp.StatusCode)
			} else {
				// 400错误，读取详细错误信息
				body, _ := io.ReadAll(resp.Body)
				var errorResp map[string]interface{}
				json.Unmarshal(body, &errorResp)
				resp.Body.Close()

				logger.Infof("Still waiting for authentication... (elapsed: %v)", time.Since(start))
				if errorResp != nil {
					logger.Infof("Error details: %+v", errorResp)
				}
			}
		}
	}
}

// generatePKCE 生成PKCE code verifier和code challenge
func (c *SimpleTodoClient) generatePKCE() (string, string) {
	// 生成随机code verifier (43-128字符)
	// 使用更少的字节以确保合适的长度
	bytes := make([]byte, 32)
	rand.Read(bytes)
	codeVerifier := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)

	// 确保长度在43-128字符之间
	if len(codeVerifier) < 43 {
		// 如果太短，增加长度
		extraBytes := make([]byte, 16)
		rand.Read(extraBytes)
		codeVerifier += base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(extraBytes)
	}
	if len(codeVerifier) > 128 {
		// 如果太长，截断
		codeVerifier = codeVerifier[:128]
	}

	// 生成code challenge (SHA256 + base64url)
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])

	return codeVerifier, codeChallenge
}

// buildAuthURLWithPKCE 构建带PKCE的授权URL
func (c *SimpleTodoClient) buildAuthURLWithPKCE(codeChallenge string) string {
	params := url.Values{}
	params.Add("client_id", c.authConfig.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", "http://localhost:8080/callback")
	params.Add("scope", "https://graph.microsoft.com/Tasks.ReadWrite https://graph.microsoft.com/User.Read offline_access")
	params.Add("state", "12345") // 简单的state值
	params.Add("code_challenge", codeChallenge)
	params.Add("code_challenge_method", "S256")

	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		c.authConfig.TenantID, params.Encode())
}

// exchangeCodeForTokenWithPKCE 使用PKCE交换访问令牌
func (c *SimpleTodoClient) exchangeCodeForTokenWithPKCE(ctx context.Context, code, codeVerifier string) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.authConfig.TenantID)

	data := url.Values{}
	data.Set("client_id", c.authConfig.ClientID)
	data.Set("client_secret", c.authConfig.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", "http://localhost:8080/callback")
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp map[string]interface{}
		json.Unmarshal(body, &errorResp)
		return "", fmt.Errorf("token request failed with status: %d, error: %+v", resp.StatusCode, errorResp)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("received empty access token")
	}

	// 缓存token以备将来使用
	refreshToken := tokenResp.RefreshToken
	if err := c.saveCachedToken(tokenResp.AccessToken, refreshToken, tokenResp.ExpiresIn); err != nil {
		logger.Infof("Warning: Failed to cache token: %v", err)
	}

	return tokenResp.AccessToken, nil
}

// buildAuthURL 构建授权URL
func (c *SimpleTodoClient) buildAuthURL() string {
	params := url.Values{}
	params.Add("client_id", c.authConfig.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", "http://localhost:8080/callback")
	params.Add("scope", "Tasks.ReadWrite User.Read offline_access")
	params.Add("state", "12345") // 简单的state值

	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		c.authConfig.TenantID, params.Encode())
}

// extractAuthCode 从回调URL中提取授权码
func (c *SimpleTodoClient) extractAuthCode(callbackURL string) (string, error) {
	parsedURL, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("invalid callback URL: %v", err)
	}

	code := parsedURL.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("authorization code not found in callback URL")
	}

	return code, nil
}

// exchangeCodeForToken 使用授权码交换访问令牌
func (c *SimpleTodoClient) exchangeCodeForToken(ctx context.Context, code string) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.authConfig.TenantID)

	data := url.Values{}
	data.Set("client_id", c.authConfig.ClientID)
	data.Set("client_secret", c.authConfig.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", "http://localhost:8080/callback")
	data.Set("scope", "https://graph.microsoft.com/Tasks.ReadWrite https://graph.microsoft.com/User.Read offline_access")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 读取错误响应以获取更多调试信息
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return "", fmt.Errorf("token request failed with status: %d, error: %+v", resp.StatusCode, errorResp)
		}
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("received empty access token")
	}

	// 缓存token以备将来使用
	refreshToken := tokenResp.RefreshToken
	if err := c.saveCachedToken(tokenResp.AccessToken, refreshToken, tokenResp.ExpiresIn); err != nil {
		logger.Infof("Warning: Failed to cache token: %v", err)
	}

	return tokenResp.AccessToken, nil
}

// makeAPIRequest 发送 API 请求
func (c *SimpleTodoClient) makeAPIRequest(ctx context.Context, method, endpoint string, requestBody interface{}) (*http.Response, error) {
	// 获取访问令牌
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// 准备请求
	url := c.baseURL + endpoint
	var body *bytes.Buffer

	if requestBody != nil {
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		body = bytes.NewBuffer(jsonBody)
	} else {
		body = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	return resp, nil
}

// makeAPIRequestWithRetry 发起带重试机制的API请求
func (c *SimpleTodoClient) makeAPIRequestWithRetry(ctx context.Context, method, endpoint string, requestBody interface{}, maxRetries int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			logger.Infof("Retrying API request (attempt %d/%d): %s %s", attempt+1, maxRetries, method, endpoint)
			// 指数退避策略
			backoffDuration := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		resp, err := c.makeAPIRequest(ctx, method, endpoint, requestBody)
		if err != nil {
			lastErr = err
			logger.Infof("API request failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
			continue
		}

		// 检查是否是服务器错误，如果是则重试
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error with status: %d", resp.StatusCode)
			logger.Infof("Server error (attempt %d/%d): %d", attempt+1, maxRetries, resp.StatusCode)
			continue
		}

		// 成功或客户端错误（4xx），直接返回
		return resp, nil
	}

	return nil, fmt.Errorf("API request failed after %d attempts: %v", maxRetries, lastErr)
}

// TestConnection 测试连接到 Microsoft Graph API
func (c *SimpleTodoClient) TestConnection() error {
	logger.Infof("Testing Microsoft Graph connection with Tenant ID: %s, Client ID: %s", c.authConfig.TenantID, c.authConfig.ClientID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用委托权限的 /me 端点
	resp, err := c.makeAPIRequest(ctx, "GET", "/me", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Microsoft Graph API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 读取错误响应以获取更多调试信息
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("API request failed with status: %d, error: %+v", resp.StatusCode, errorResp)
		}
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// 解析响应
	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	displayName, ok := user["displayName"].(string)
	if !ok {
		return fmt.Errorf("connected but received invalid user response")
	}

	logger.Infof("Successfully connected to Microsoft Graph API as user: %s", displayName)
	return nil
}

// GetOrCreateTaskList 获取或创建任务列表
func (c *SimpleTodoClient) GetOrCreateTaskList(listName string) (string, error) {
	logger.Infof("Getting or creating task list: %s", listName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用委托权限获取现有的任务列表
	resp, err := c.makeAPIRequest(ctx, "GET", "/me/todo/lists", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get task lists: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get task lists with status: %d", resp.StatusCode)
	}

	// 解析响应
	var response struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode task lists response: %v", err)
	}

	// 查找是否已存在指定名称的列表
	for _, list := range response.Value {
		if list.DisplayName == listName {
			logger.Infof("Found existing task list '%s' with ID: %s", listName, list.ID)
			return list.ID, nil
		}
	}

	// 如果没有找到，创建新的任务列表
	logger.Infof("Creating new task list: %s", listName)

	newList := map[string]interface{}{
		"displayName": listName,
	}

	resp, err = c.makeAPIRequest(ctx, "POST", "/me/todo/lists", newList)
	if err != nil {
		return "", fmt.Errorf("failed to create task list: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create task list with status: %d", resp.StatusCode)
	}

	// 解析创建的列表响应
	var createdList struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createdList); err != nil {
		return "", fmt.Errorf("failed to decode created list response: %v", err)
	}

	logger.Infof("Successfully created task list '%s' with ID: %s", listName, createdList.ID)
	return createdList.ID, nil
}

// CreateTaskWithDetails 创建带详细信息的任务
func (c *SimpleTodoClient) CreateTaskWithDetails(title, description, listID string, dueTime, reminderTime time.Time, importance int, timezone string) error {
	logger.Infof("Creating task: %s", title)
	if description != "" {
		logger.Infof("Task description: %s", description)
	}
	if !dueTime.IsZero() {
		logger.Infof("Due time: %s", dueTime.Format("2006-01-02 15:04"))
	}
	if !reminderTime.IsZero() {
		logger.Infof("Reminder time: %s", reminderTime.Format("2006-01-02 15:04"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建任务请求体
	newTask := map[string]interface{}{
		"title": title,
	}

	if description != "" {
		newTask["body"] = map[string]interface{}{
			"content":     description,
			"contentType": "text",
		}
	}

	// 设置截止时间
	if !dueTime.IsZero() {
		// 使用UTC标准化处理：从UTC时间转换为目标时区
		// 这避免了Windows系统的时区数据库问题和双重时区转换
		targetTime := timezonepkg.ConvertToTargetTimezone(dueTime, timezone)
		formattedTime := timezonepkg.FormatTimeForGraphAPI(dueTime, timezone)

		newTask["dueDateTime"] = map[string]interface{}{
			"dateTime": formattedTime,
			"timeZone": timezone,
		}
		logger.Infof("设置截止时间: %s (原始UTC: %s, 目标时区: %s)",
			targetTime.Format("2006-01-02 15:04:05"),
			dueTime.UTC().Format("2006-01-02 15:04:05"),
			timezone)
	}

	// 设置提醒时间
	if !reminderTime.IsZero() {
		// 使用UTC标准化处理：从UTC时间转换为目标时区
		// 保持与dueTime处理的一致性
		targetReminderTime := timezonepkg.ConvertToTargetTimezone(reminderTime, timezone)
		formattedReminderTime := timezonepkg.FormatTimeForGraphAPI(reminderTime, timezone)

		logger.Infof("设置提醒时间: %s (原始UTC: %s, 目标时区: %s)",
			targetReminderTime.Format("2006-01-02 15:04:05"),
			reminderTime.UTC().Format("2006-01-02 15:04:05"),
			timezone)

		// 使用标准化的时间格式，与dueDateTime保持一致
		newTask["reminderDateTime"] = map[string]interface{}{
			"dateTime": formattedReminderTime,
			"timeZone": timezone,
		}
	} else {
		logger.Warnf("提醒时间为空，Microsoft Todo不会创建提醒")
	}

	// 设置重要性
	switch importance {
	case 1: // 低优先级
		newTask["importance"] = "low"
	case 5: // 中等优先级
		newTask["importance"] = "normal"
	case 9: // 高优先级
		newTask["importance"] = "high"
	default:
		newTask["importance"] = "normal"
	}

	// 发送创建任务请求
	endpoint := fmt.Sprintf("/me/todo/lists/%s/tasks", listID)
	resp, err := c.makeAPIRequest(ctx, "POST", endpoint, newTask)
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// 读取错误响应以获取更多调试信息
		body, _ := io.ReadAll(resp.Body)
		var errorResp map[string]interface{}
		json.Unmarshal(body, &errorResp)
		return fmt.Errorf("failed to create task with status: %d, error: %+v", resp.StatusCode, errorResp)
	}

	// 解析创建的任务响应
	var createdTask struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createdTask); err != nil {
		return fmt.Errorf("failed to decode created task response: %v", err)
	}

	logger.Infof("Successfully created task '%s' with ID: %s", title, createdTask.ID)
	return nil
}

// CreateTask 创建任务（保持向后兼容）
func (c *SimpleTodoClient) CreateTask(title, description, listID string) error {
	return c.CreateTaskWithDetails(title, description, listID, time.Time{}, time.Time{}, 5, "UTC")
}

// GetServerInfo 获取服务器信息
func (c *SimpleTodoClient) GetServerInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})
	info["service"] = "Microsoft Graph API"
	info["api"] = "To Do Lists"

	// 测试连接以获取真实状态
	if err := c.TestConnection(); err != nil {
		info["status"] = "Connection Failed"
		info["error"] = err.Error()
	} else {
		info["status"] = "Connected"
	}

	return info, nil
}

// isExpired 检查token是否过期
func (t *TokenData) isExpired() bool {
	// 提前5分钟过期以确保安全
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// getTokenCachePath 获取token缓存文件路径
func (c *SimpleTodoClient) getTokenCachePath() string {
	// 在用户主目录下创建缓存目录
	cacheDir := filepath.Join(os.Getenv("USERPROFILE"), ".to_icalendar")
	os.MkdirAll(cacheDir, 0700)

	// 使用client_id作为文件名的一部分确保唯一性
	filename := fmt.Sprintf("token_%s.json", strings.ReplaceAll(c.authConfig.ClientID, "-", "_"))
	return filepath.Join(cacheDir, filename)
}

// loadCachedToken 从缓存加载token
func (c *SimpleTodoClient) loadCachedToken() (*TokenData, error) {
	cachePath := c.getTokenCachePath()

	// 检查文件是否存在
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no cached token file")
	}

	// 读取文件
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %v", err)
	}

	// 解析token
	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse cached token: %v", err)
	}

	return &token, nil
}

// saveCachedToken 保存token到缓存
func (c *SimpleTodoClient) saveCachedToken(accessToken, refreshToken string, expiresIn int) error {
	tokenData := &TokenData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
		TokenType:    "Bearer",
	}

	cachePath := c.getTokenCachePath()

	// 序列化token
	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize token: %v", err)
	}

	// 写入文件，设置安全权限
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	logger.Infof("Token cached successfully to: %s", cachePath)
	return nil
}

// refreshAccessToken 刷新访问令牌
func (c *SimpleTodoClient) refreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.authConfig.TenantID)

	data := url.Values{}
	data.Set("client_id", c.authConfig.ClientID)
	data.Set("client_secret", c.authConfig.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create refresh request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp map[string]interface{}
		json.Unmarshal(body, &errorResp)
		return "", fmt.Errorf("token refresh failed with status: %d, error: %+v", resp.StatusCode, errorResp)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode refresh response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("received empty access token from refresh")
	}

	// 如果返回了新的refresh token，更新缓存
	if tokenResp.RefreshToken != "" {
		if err := c.saveCachedToken(tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn); err != nil {
			logger.Infof("Warning: Failed to cache refreshed token: %v", err)
		}
	}

	return tokenResp.AccessToken, nil
}

// clearCachedToken 清除缓存的token
func (c *SimpleTodoClient) clearCachedToken() error {
	cachePath := c.getTokenCachePath()
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %v", err)
	}
	logger.Infof("Cached token cleared")
	return nil
}

// TaskInfo 表示从 Microsoft Todo 查询到的任务信息
type TaskInfo struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"body"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdDateTime"`
	DueDateTime   string    `json:"dueDateTime"`
	Importance    string    `json:"importance"`
	IsCompleted   bool      `json:"completed"`
}

// generateQueryCacheKey 生成查询缓存键
func (c *SimpleTodoClient) generateQueryCacheKey(listName, queryType, params string) string {
	data := fmt.Sprintf("%s|%s|%s", listName, queryType, params)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// QueryIncompleteTasks 查询未完成的任务
func (c *SimpleTodoClient) QueryIncompleteTasks(listName string) ([]TaskInfo, error) {
	logger.Infof("Querying incomplete tasks in list: %s", listName)

	// 生成缓存键
	cacheKey := c.generateQueryCacheKey(listName, "incomplete", "")

	// 尝试从缓存获取结果
	if tasks, found := c.queryCache.Get(cacheKey); found {
		logger.Infof("Cache hit for incomplete tasks query: %s (%d tasks)", listName, len(tasks))
		return tasks, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 首先获取任务列表ID
	listID, err := c.GetOrCreateTaskList(listName)
	if err != nil {
		return nil, fmt.Errorf("failed to get task list: %v", err)
	}

	// 构建查询参数，获取所有任务，然后在客户端过滤
	// 简化查询，避免复杂的 OData 过滤语法
	endpoint := fmt.Sprintf("/me/todo/lists/%s/tasks", listID)
	logger.Infof("Query endpoint: %s", endpoint) // 调试信息

	resp, err := c.makeAPIRequestWithRetry(ctx, "GET", endpoint, nil, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			logger.Infof("Failed to unmarshal error response: %v, raw body: %s", err, string(body))
		}

		// 记录详细的错误信息用于调试
		logger.Infof("Query failed with status: %d, endpoint: %s", resp.StatusCode, endpoint)
		logger.Infof("Error response body: %s", string(body))

		// 检查特定的错误类型
		if errorResp != nil {
			if error, ok := errorResp["error"].(map[string]interface{}); ok {
				if code, ok := error["code"].(string); ok {
					switch code {
					case "Request_ResourceNotFound":
						return nil, fmt.Errorf("task list not found: %s", listName)
					case "AuthenticationError":
						return nil, fmt.Errorf("authentication failed, please re-authenticate")
					case "InvalidFilterClause":
						return nil, fmt.Errorf("invalid OData filter clause in query: %s", endpoint)
					}
				}
			}
		}

		return nil, fmt.Errorf("failed to query tasks with status: %d, error: %+v", resp.StatusCode, errorResp)
	}

	// 解析响应
	var response struct {
		Value []struct {
			ID            string                 `json:"id"`
			Title         string                 `json:"title"`
			Body          map[string]interface{} `json:"body,omitempty"`
			Status        string                 `json:"status"`
			CreatedDateTime string               `json:"createdDateTime"`
			DueDateTime   map[string]interface{} `json:"dueDateTime,omitempty"`
			Importance    string                 `json:"importance"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode tasks response: %v", err)
	}

	// 转换为TaskInfo结构
	var tasks []TaskInfo
	for _, task := range response.Value {
		taskInfo := TaskInfo{
			ID:          task.ID,
			Title:       task.Title,
			Status:      task.Status,
			CreatedAt:   parseTime(task.CreatedDateTime),
			Importance:  task.Importance,
			IsCompleted: task.Status == "completed",
		}

		// 提取描述信息
		if task.Body != nil {
			if content, ok := task.Body["content"].(string); ok {
				taskInfo.Description = content
			}
		}

		// 提取截止时间
		if task.DueDateTime != nil {
			if dateTime, ok := task.DueDateTime["dateTime"].(string); ok {
				taskInfo.DueDateTime = dateTime
			}
		}

			// 只包含未完成的任务
		if !taskInfo.IsCompleted {
			tasks = append(tasks, taskInfo)
		}
	}

	logger.Infof("Found %d incomplete tasks in list '%s' (filtered from %d total)", len(tasks), listName, len(response.Value))

	// 将结果添加到缓存
	c.queryCache.Set(cacheKey, tasks)
	logger.Infof("Cached incomplete tasks query result: %s (%d tasks)", listName, len(tasks))

	return tasks, nil
}

// QueryTasksByTitle 根据标题模糊查询任务
func (c *SimpleTodoClient) QueryTasksByTitle(listName, titleKeyword string, incompleteOnly bool) ([]TaskInfo, error) {
	logger.Infof("Querying tasks by keyword '%s' in list: %s (incomplete only: %t)", titleKeyword, listName, incompleteOnly)

	// 生成缓存键
	incompleteStr := "false"
	if incompleteOnly {
		incompleteStr = "true"
	}
	cacheKey := c.generateQueryCacheKey(listName, "by_title", fmt.Sprintf("%s|%s", titleKeyword, incompleteStr))

	// 尝试从缓存获取结果
	if tasks, found := c.queryCache.Get(cacheKey); found {
		logger.Infof("Cache hit for tasks by title query: %s - %s (%d tasks)", listName, titleKeyword, len(tasks))
		return tasks, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取任务列表ID
	listID, err := c.GetOrCreateTaskList(listName)
	if err != nil {
		return nil, fmt.Errorf("failed to get task list: %v", err)
	}

	// 构建查询参数 - 使用最简单的查询方式
	endpoint := fmt.Sprintf("/me/todo/lists/%s/tasks", listID)
	logger.Infof("Query endpoint: %s", endpoint) // 调试信息

	resp, err := c.makeAPIRequestWithRetry(ctx, "GET", endpoint, nil, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by title: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			logger.Infof("Failed to unmarshal error response: %v, raw body: %s", err, string(body))
		}

		// 记录详细的错误信息用于调试
		logger.Infof("Query failed with status: %d, endpoint: %s", resp.StatusCode, endpoint)
		logger.Infof("Error response body: %s", string(body))

		// 检查特定的错误类型
		if errorResp != nil {
			if error, ok := errorResp["error"].(map[string]interface{}); ok {
				if code, ok := error["code"].(string); ok {
					switch code {
					case "Request_ResourceNotFound":
						return nil, fmt.Errorf("task list not found: %s", listName)
					case "AuthenticationError":
						return nil, fmt.Errorf("authentication failed, please re-authenticate")
					case "InvalidFilterClause":
						return nil, fmt.Errorf("invalid OData filter clause in query")
					}
				}
			}
		}

		return nil, fmt.Errorf("failed to query tasks by title with status: %d, error: %+v", resp.StatusCode, errorResp)
	}

	// 解析响应
	var response struct {
		Value []struct {
			ID            string                 `json:"id"`
			Title         string                 `json:"title"`
			Body          map[string]interface{} `json:"body,omitempty"`
			Status        string                 `json:"status"`
			CreatedDateTime string               `json:"createdDateTime"`
			DueDateTime   map[string]interface{} `json:"dueDateTime,omitempty"`
			Importance    string                 `json:"importance"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode tasks response: %v", err)
	}

	// 转换为TaskInfo结构
	var tasks []TaskInfo
	for _, task := range response.Value {
		taskInfo := TaskInfo{
			ID:          task.ID,
			Title:       task.Title,
			Status:      task.Status,
			CreatedAt:   parseTime(task.CreatedDateTime),
			Importance:  task.Importance,
			IsCompleted: task.Status == "completed",
		}

		// 提取描述信息
		if task.Body != nil {
			if content, ok := task.Body["content"].(string); ok {
				taskInfo.Description = content
			}
		}

		// 提取截止时间
		if task.DueDateTime != nil {
			if dateTime, ok := task.DueDateTime["dateTime"].(string); ok {
				taskInfo.DueDateTime = dateTime
			}
		}

		// 客户端过滤条件
		shouldInclude := true

		// 过滤已完成任务
		if incompleteOnly && taskInfo.IsCompleted {
			shouldInclude = false
		}

		// 按标题关键词过滤
		if titleKeyword != "" {
			titleLower := strings.ToLower(taskInfo.Title)
			keywordLower := strings.ToLower(titleKeyword)
			if !strings.Contains(titleLower, keywordLower) {
				shouldInclude = false
			}
		}

		if shouldInclude {
			tasks = append(tasks, taskInfo)
		}
	}

	logger.Infof("Found %d tasks matching '%s' in list '%s' (filtered from %d total)", len(tasks), titleKeyword, listName, len(response.Value))

	// 将结果添加到缓存
	c.queryCache.Set(cacheKey, tasks)
	logger.Infof("Cached tasks by title query result: %s - %s (%d tasks)", listName, titleKeyword, len(tasks))

	return tasks, nil
}

// GetQueryCacheStats 获取查询缓存统计信息
func (c *SimpleTodoClient) GetQueryCacheStats() map[string]interface{} {
	if c.queryCache == nil {
		return map[string]interface{}{
			"cache_enabled": false,
		}
	}

	stats := c.queryCache.GetStats()
	stats["cache_enabled"] = true
	return stats
}

// ClearQueryCache 清空查询缓存
func (c *SimpleTodoClient) ClearQueryCache() {
	if c.queryCache != nil {
		c.queryCache.Clear()
		logger.Infof("Query cache cleared")
	}
}

// parseTime 解析时间字符串
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}

	// Microsoft Graph API 返回的时间格式是 RFC3339 格式
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t
	}

	// 尝试其他可能的格式
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	return time.Time{}
}
