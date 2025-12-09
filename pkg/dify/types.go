package dify

import (
	"time"

	"github.com/allanpk716/to_icalendar/pkg/models"
)

// APIRequest represents a generic API request structure
type APIRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`          // 输入参数
	Query          string                 `json:"query"`           // 查询内容
	ResponseMode   string                 `json:"response_mode"`    // 响应模式：blocking/streaming
	User           string                 `json:"user"`            // 用户标识
	AutoGenerateName bool                 `json:"auto_generate_name"` // 是否自动生成名称
	ConversationID string                 `json:"conversation_id,omitempty"` // 对话ID（可选）
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	File     []byte `json:"file"`               // 文件内容
	User     string `json:"user"`               // 用户标识
	Purpose  string `json:"purpose,omitempty"`  // 文件用途（可选）
}

// FileUploadResponse represents a file upload response
type FileUploadResponse struct {
	ID        string    `json:"id"`         // 文件ID
	Name      string    `json:"name"`       // 文件名
	Size      int64     `json:"size"`       // 文件大小
	MimeType  string    `json:"mime_type"`  // MIME类型
	URL       string    `json:"url"`        // 文件URL
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// ChatMessageRequest represents a chat message request with file support
type ChatMessageRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`                   // 输入参数
	Query          string                 `json:"query"`                    // 查询内容
	ResponseMode   string                 `json:"response_mode"`             // 响应模式
	User           string                 `json:"user"`                     // 用户标识
	AutoGenerateName bool                 `json:"auto_generate_name"`       // 是否自动生成名称
	ConversationID string                 `json:"conversation_id,omitempty"` // 对话ID（可选）
	Files          []models.DifyFile      `json:"files,omitempty"`          // 文件列表（可选）
}

// ChatMessageResponse represents a chat message response
type ChatMessageResponse struct {
	MessageID      string                 `json:"message_id"`       // 消息ID
	ConversationID string                 `json:"conversation_id"`  // 对话ID
	Answer         string                 `json:"answer"`           // AI回答内容
	CreatedAt      time.Time              `json:"created_at"`       // 创建时间
	Metadata       map[string]interface{} `json:"metadata,omitempty"` // 元数据
}

// StreamEvent represents a streaming event response
type StreamEvent struct {
	Event   string      `json:"event"`   // 事件类型
	MessageID string    `json:"message_id,omitempty"` // 消息ID（某些事件包含）
	TaskID  string      `json:"task_id,omitempty"`    // 任务ID（某些事件包含）
	Data    interface{} `json:"data,omitempty"`       // 事件数据
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Code    string `json:"code"`    // 错误代码
	Message string `json:"message"` // 错误消息
	Status  int    `json:"status"`  // HTTP状态码
}

// TaskExtractionPrompt defines the prompt for task extraction
const TaskExtractionPrompt = `请分析以下内容并提取任务信息。如果内容包含任务相关事项，请按照以下JSON格式返回：
{
  "title": "任务标题",
  "description": "任务描述（可选）",
  "date": "YYYY-MM-DD",
  "time": "HH:MM",
  "remind_before": "15m（可选，默认15分钟）",
  "priority": "low/medium/high（可选，默认medium）",
  "list": "任务列表名称（可选，默认Default）",
  "confidence": 0.95
}

如果内容不是任务相关，请返回：
{
  "error": "未识别到任务信息",
  "confidence": 0.1,
  "summary": "内容摘要"
}

请确保日期时间格式准确，优先级使用明确的词汇。`

// ImageAnalysisPrompt defines the prompt for image analysis
const ImageAnalysisPrompt = `请分析这张图片中的文字内容。如果是任务相关的截图（如会议通知、待办事项、日历事件等），请提取任务信息并按照以下JSON格式返回：
{
  "title": "任务标题",
  "description": "详细描述",
  "date": "YYYY-MM-DD",
  "time": "HH:MM",
  "remind_before": "15m（可选）",
  "priority": "low/medium/high（可选）",
  "list": "任务列表名称（可选）",
  "confidence": 0.95,
  "original_text": "图片中的原始文字"
}

如果图片不包含任务信息，请返回：
{
  "error": "图片中未识别到任务信息",
  "confidence": 0.1,
  "summary": "图片内容描述",
  "original_text": "图片中的文字内容"
}

请仔细识别图片中的所有文字，包括手写文字，并准确提取时间信息。`

// ProcessingOptions defines options for content processing
type ProcessingOptions struct {
	MaxRetries      int           `json:"max_retries"`       // 最大重试次数
	RetryDelay      time.Duration `json:"retry_delay"`       // 重试延迟
	Timeout         time.Duration `json:"timeout"`           // 处理超时时间
	EnableOCR       bool          `json:"enable_ocr"`        // 是否启用OCR
	ConfidenceThreshold float64   `json:"confidence_threshold"` // 置信度阈值
	DefaultList     string        `json:"default_list"`      // 默认任务列表
	DefaultPriority string        `json:"default_priority"`  // 默认优先级
	DefaultRemindBefore string    `json:"default_remind_before"` // 默认提前提醒时间
}

// DefaultProcessingOptions returns default processing options
func DefaultProcessingOptions() *ProcessingOptions {
	return &ProcessingOptions{
		MaxRetries:         3,
		RetryDelay:         2 * time.Second,
		Timeout:            30 * time.Second,
		EnableOCR:          true,
		ConfidenceThreshold: 0.7,
		DefaultList:        "Default",
		DefaultPriority:    "medium",
		DefaultRemindBefore: "15m", // 默认15分钟提醒
	}
}

// ValidationResult represents the result of content validation
type ValidationResult struct {
	IsValid   bool    `json:"is_valid"`    // 是否有效
	ErrorType string  `json:"error_type"`  // 错误类型
	Message   string  `json:"message"`     // 验证消息
	Score     float64 `json:"score"`       // 验证分数（0-1）
}

// ContentType represents the type of content being processed
type ContentType string

const (
	ContentTypeText      ContentType = "text"       // 纯文本内容
	ContentTypeImage     ContentType = "image"      // 图片内容
	ContentTypeMixed     ContentType = "mixed"      // 混合内容
	ContentTypeUnknown   ContentType = "unknown"    // 未知类型
)

// ProcessingRequest represents a request to process content
type ProcessingRequest struct {
	Content     *models.ClipboardContent `json:"content"`       // 剪贴板内容
	Options     *ProcessingOptions       `json:"options"`       // 处理选项
	UserID      string                   `json:"user_id"`       // 用户ID
	RequestType ContentType              `json:"request_type"`  // 请求类型
}

// ProcessingResponse represents the response from content processing
type ProcessingResponse struct {
	Success        bool                     `json:"success"`         // 是否成功
	Reminder       *models.Reminder         `json:"reminder,omitempty"` // 生成的提醒事项
	ParsedInfo     *models.ParsedTaskInfo   `json:"parsed_info,omitempty"` // 解析信息
	Validation     *ValidationResult        `json:"validation,omitempty"` // 验证结果
	ProcessingTime time.Duration            `json:"processing_time"` // 处理时间
	RequestID      string                   `json:"request_id"`     // 请求ID
	Timestamp      time.Time                `json:"timestamp"`      // 时间戳
	ErrorMessage   string                   `json:"error_message,omitempty"` // 错误信息
}