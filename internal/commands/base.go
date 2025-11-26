package commands

// BaseCommand 基础命令实现
type BaseCommand struct {
	name        string
	description string
}

// NewBaseCommand 创建基础命令
func NewBaseCommand(name, description string) *BaseCommand {
	return &BaseCommand{
		name:        name,
		description: description,
	}
}

// GetName 获取命令名称
func (c *BaseCommand) GetName() string {
	return c.name
}

// GetDescription 获取命令描述
func (c *BaseCommand) GetDescription() string {
	return c.description
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}, metadata map[string]interface{}) *CommandResponse {
	return &CommandResponse{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(err error) *CommandResponse {
	return &CommandResponse{
		Success: false,
		Error:   err.Error(),
	}
}