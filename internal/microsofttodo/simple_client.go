package microsofttodo

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// SimpleTodoClient 简化的 Microsoft Todo 客户端
type SimpleTodoClient struct {
	authConfig *AuthConfig
	httpClient *http.Client
	baseURL    string
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
	}, nil
}

// getAccessToken 获取访问令牌
func (c *SimpleTodoClient) getAccessToken(ctx context.Context) (string, error) {
	// 首先尝试从缓存加载token
	if token, err := c.loadCachedToken(); err == nil && token != nil {
		if !token.isExpired() {
			log.Printf("Using cached access token")
			return token.AccessToken, nil
		}

		// token过期但refresh token有效，尝试刷新
		if token.RefreshToken != "" {
			log.Printf("Access token expired, attempting to refresh...")
			if newToken, err := c.refreshAccessToken(ctx, token.RefreshToken); err == nil {
				log.Printf("Token refreshed successfully")
				return newToken, nil
			}
			log.Printf("Token refresh failed: %v", err)
		}
	}

	// 如果没有有效token或refresh失败，进行交互式认证
	log.Printf("No valid cached token found, starting interactive authentication")
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
	log.Printf("Authentication successful, caching token for future use")
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
			log.Printf("Checking authentication status... (elapsed: %v)", time.Since(start))
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
				log.Printf("Authentication successful! (elapsed: %v)", time.Since(start))
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
				log.Printf("Unexpected status code: %d", resp.StatusCode)
			} else {
				// 400错误，读取详细错误信息
				body, _ := io.ReadAll(resp.Body)
				var errorResp map[string]interface{}
				json.Unmarshal(body, &errorResp)
				resp.Body.Close()

				log.Printf("Still waiting for authentication... (elapsed: %v)", time.Since(start))
				if errorResp != nil {
					log.Printf("Error details: %+v", errorResp)
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
		log.Printf("Warning: Failed to cache token: %v", err)
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
		log.Printf("Warning: Failed to cache token: %v", err)
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

// TestConnection 测试连接到 Microsoft Graph API
func (c *SimpleTodoClient) TestConnection() error {
	log.Printf("Testing Microsoft Graph connection with Tenant ID: %s, Client ID: %s", c.authConfig.TenantID, c.authConfig.ClientID)

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

	log.Printf("Successfully connected to Microsoft Graph API as user: %s", displayName)
	return nil
}

// GetOrCreateTaskList 获取或创建任务列表
func (c *SimpleTodoClient) GetOrCreateTaskList(listName string) (string, error) {
	log.Printf("Getting or creating task list: %s", listName)

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
			log.Printf("Found existing task list '%s' with ID: %s", listName, list.ID)
			return list.ID, nil
		}
	}

	// 如果没有找到，创建新的任务列表
	log.Printf("Creating new task list: %s", listName)

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

	log.Printf("Successfully created task list '%s' with ID: %s", listName, createdList.ID)
	return createdList.ID, nil
}

// CreateTask 创建任务
func (c *SimpleTodoClient) CreateTask(title, description, listID string) error {
	log.Printf("Creating task: %s", title)
	if description != "" {
		log.Printf("Task description: %s", description)
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

	// 发送创建任务请求
	endpoint := fmt.Sprintf("/me/todo/lists/%s/tasks", listID)
	resp, err := c.makeAPIRequest(ctx, "POST", endpoint, newTask)
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create task with status: %d", resp.StatusCode)
	}

	// 解析创建的任务响应
	var createdTask struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createdTask); err != nil {
		return fmt.Errorf("failed to decode created task response: %v", err)
	}

	log.Printf("Successfully created task '%s' with ID: %s", title, createdTask.ID)
	return nil
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

	log.Printf("Token cached successfully to: %s", cachePath)
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
			log.Printf("Warning: Failed to cache refreshed token: %v", err)
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
	log.Printf("Cached token cleared")
	return nil
}
