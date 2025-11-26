package commands

// CommandExecutor 统一命令执行器接口
type CommandExecutor interface {
	// InitConfig 初始化配置
	InitConfig() (*ConfigResult, error)

	// CleanCache 清理缓存
	CleanCache(options *CleanupOptions) (*CleanupResult, error)

	// ProcessClipboard 处理剪贴板并上传
	ProcessClipboard() (*ProcessClipboardResult, error)
}

// ConfigResult 配置操作结果
type ConfigResult struct {
	Success          bool   `json:"success"`
	ConfigDir        string `json:"config_dir"`
	ServerConfig     string `json:"server_config"`
	ReminderTemplate string `json:"reminder_template"`
	Message          string `json:"message"`
}

// CleanupOptions 清理选项
type CleanupOptions struct {
	All          bool   `json:"all"`
	Tasks        bool   `json:"tasks"`
	Images       bool   `json:"images"`
	ImageHashes  bool   `json:"image_hashes"`
	Temp         bool   `json:"temp"`
	Generated    bool   `json:"generated"`
	DryRun       bool   `json:"dry_run"`
	Force        bool   `json:"force"`
	OlderThan    string `json:"older_than"`
	ClearAll     bool   `json:"clear_all"`
}

// CleanupResult 清理结果
type CleanupResult struct {
	Success    bool            `json:"success"`
	TotalFiles int64           `json:"total_files"`
	TotalSize  int64           `json:"total_size"`
	FilesByType map[string]int64 `json:"files_by_type"`
	Skipped    bool            `json:"skipped"`
	Message    string          `json:"message"`
}

// ProcessClipboardResult 剪贴板处理结果
type ProcessClipboardResult struct {
	Success     bool    `json:"success"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Message     string  `json:"message,omitempty"`
}