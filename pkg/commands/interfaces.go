package commands

import (
	"context"

	"github.com/allanpk716/to_icalendar/pkg/services"
)

// CommandExecutor 命令执行器接口
type CommandExecutor interface {
	Execute(ctx context.Context, req *CommandRequest) (*CommandResponse, error)
	GetName() string
	GetDescription() string
	Validate(args []string) error
}

// CommandRequest 命令请求
type CommandRequest struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args"`
	Options map[string]interface{} `json:"options"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// CommandResponse 命令响应
type CommandResponse struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceContainer 服务容器接口
type ServiceContainer interface {
	GetConfigService() services.ConfigService
	GetCacheService() services.CacheService
	GetClipboardService() services.ClipboardService
	GetCleanupService() services.CleanupService
	GetTodoService() services.TodoService
	GetDifyService() services.DifyService
	GetLogger() interface{}
}


// SuccessResponseWithData 创建带数据的成功响应
func SuccessResponseWithData(data interface{}) *CommandResponse {
	return &CommandResponse{
		Success: true,
		Data:    data,
	}
}