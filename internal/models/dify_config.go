package models

import (
	"fmt"
	"time"
)

// DifyConfig represents the configuration for Dify API integration
type DifyConfig struct {
	APIEndpoint string `yaml:"api_endpoint"` // Dify API 端点
	APIKey      string `yaml:"api_key"`      // Dify API 密钥
	Model       string `yaml:"model"`        // Dify 模型名称
	MaxTokens   int    `yaml:"max_tokens"`   // 最大token数量
	Timeout     int    `yaml:"timeout"`      // 请求超时时间（秒）
}

// Validate validates the Dify configuration
func (c *DifyConfig) Validate() error {
	if c.APIEndpoint == "" {
		return fmt.Errorf("Dify API endpoint is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("Dify API key is required")
	}

	if c.APIKey == "YOUR_DIFY_API_KEY" {
		return fmt.Errorf("please configure a valid Dify API key")
	}

	if c.Model == "" {
		return fmt.Errorf("Dify model name is required")
	}

	if c.MaxTokens <= 0 || c.MaxTokens > 8000 {
		return fmt.Errorf("max_tokens must be between 1 and 8000")
	}

	if c.Timeout <= 0 || c.Timeout > 300 {
		return fmt.Errorf("timeout must be between 1 and 300 seconds")
	}

	return nil
}

// DifyRequest represents a request to the Dify API for content processing.
type DifyRequest struct {
	Inputs       map[string]interface{} `json:"inputs"`                 // 输入数据
	Query        string                  `json:"query"`                  // 查询内容
	ResponseMode string                  `json:"response_mode"`           // 响应模式 ("blocking" 或 "streaming")
	User         string                  `json:"user"`                   // 用户标识
	AutoGenerateName bool                `json:"auto_generate_name"`     // 是否自动生成名称
}

// DifyImageRequest represents a request to the Dify API for image processing.
type DifyImageRequest struct {
	Inputs       map[string]interface{} `json:"inputs"`                 // 输入数据，包含图片数据
	Query        string                  `json:"query"`                  // 查询指令
	ResponseMode string                  `json:"response_mode"`           // 响应模式
	User         string                  `json:"user"`                   // 用户标识
	AutoGenerateName bool                `json:"auto_generate_name"`     // 是否自动生成名称
	Files        []DifyFile              `json:"files,omitempty"`        // 文件列表（图片）
}

// DifyFile represents a file to be sent to Dify API.
type DifyFile struct {
	Type      string `json:"type"`      // 文件类型 ("image")
	TransferMethod string `json:"transfer_method"` // 传输方法 ("remote_url" 或 "local_file")
	URL       string `json:"url,omitempty"`        // 文件URL（当transfer_method为remote_url时）
	UploadFileID string `json:"upload_file_id,omitempty"` // 上传文件ID（当transfer_method为local_file时）
}

// DifyResponse represents the response from Dify API.
type DifyResponse struct {
	Answer      string                 `json:"answer"`                   // AI生成的回答
	MessageID   string                 `json:"message_id"`               // 消息ID
	CreatedAt   time.Time              `json:"created_at"`               // 创建时间
	ConversationID string              `json:"conversation_id,omitempty"` // 对话ID
	Metadata    map[string]interface{} `json:"metadata,omitempty"`       // 元数据
}

// DifyErrorResponse represents an error response from Dify API.
type DifyErrorResponse struct {
	Code    string `json:"code"`               // 错误代码
	Message string `json:"message"`            // 错误消息
	Status  int    `json:"status,omitempty"`   // HTTP状态码
}

// ParsedTaskInfo represents the structured task information extracted from Dify response.
type ParsedTaskInfo struct {
	Title        string    `json:"title"`                   // 任务标题
	Description  string    `json:"description,omitempty"`   // 任务描述
	Date         string    `json:"date"`                    // 任务日期 (YYYY-MM-DD)
	Time         string    `json:"time"`                    // 任务时间 (HH:MM)
	RemindBefore string    `json:"remind_before,omitempty"` // 提前提醒时间
	Priority     string    `json:"priority,omitempty"`      // 优先级
	List         string    `json:"list,omitempty"`          // 任务列表
	Confidence   float64   `json:"confidence"`              // 解析置信度 (0-1)
	OriginalText string    `json:"original_text"`           // 原始识别文本
}

// ClipboardContent represents the content read from clipboard.
type ClipboardContent struct {
	Type     ContentType `json:"type"`       // 内容类型
	Text     string      `json:"text,omitempty"` // 文字内容
	Image    []byte      `json:"image,omitempty"` // 图片数据（二进制）
	FileName string      `json:"file_name,omitempty"` // 临时文件名
}

// ContentType represents the type of clipboard content.
type ContentType string

const (
	ContentTypeText    ContentType = "text"    // 文字内容
	ContentTypeImage   ContentType = "image"   // 图片内容
	ContentTypeEmpty   ContentType = "empty"   // 空内容
	ContentTypeUnknown ContentType = "unknown" // 未知内容类型
)

// ProcessingResult represents the result of processing clipboard content.
type ProcessingResult struct {
	Success      bool           `json:"success"`        // 是否处理成功
	Reminder     *Reminder      `json:"reminder,omitempty"` // 生成的提醒事项
	ParsedInfo   *ParsedTaskInfo `json:"parsed_info,omitempty"` // 解析的任务信息
	ErrorMessage string         `json:"error_message,omitempty"` // 错误信息
	ProcessingTime time.Duration `json:"processing_time"` // 处理耗时
}