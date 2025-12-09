package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/dify"
	"github.com/allanpk716/to_icalendar/pkg/models"
	"github.com/allanpk716/to_icalendar/pkg/services"
)

// NewDifyService 创建 Dify 服务
func NewDifyService(config *models.ServerConfig, logger interface{}) services.DifyService {
	return &DifyServiceImpl{
		config: config,
		logger: logger,
	}
}

// DifyServiceImpl Dify 服务实现
type DifyServiceImpl struct {
	config *models.ServerConfig
	logger interface{}
}

// ProcessText 处理文本
func (ds *DifyServiceImpl) ProcessText(ctx context.Context, text string) (*models.DifyResponse, error) {
	if ds.config == nil {
		return nil, fmt.Errorf("配置未初始化")
	}

	// 验证 Dify 配置
	if err := ds.config.Dify.Validate(); err != nil {
		return nil, fmt.Errorf("Dify 配置验证失败: %w", err)
	}

	// 创建 Dify 客户端
	difyClient := dify.NewDifyClient(&ds.config.Dify)

	// 创建处理选项
	processingOptions := dify.DefaultProcessingOptions()
	processingOptions.DefaultRemindBefore = ds.config.Reminder.DefaultRemindBefore

	// 创建处理器
	difyProcessor := dify.NewProcessor(difyClient, "dify-service-user", processingOptions)

	// 处理文本
	processingResp, err := difyProcessor.ProcessText(ctx, text)
	if err != nil {
		return nil, err
	}

	// 转换为 models.DifyResponse
	return ds.convertProcessingResponseToDifyResponse(processingResp), nil
}

// ProcessImage 处理图片
func (ds *DifyServiceImpl) ProcessImage(ctx context.Context, imageData []byte) (*models.DifyResponse, error) {
	if ds.config == nil {
		return nil, fmt.Errorf("配置未初始化")
	}

	// 验证 Dify 配置
	if err := ds.config.Dify.Validate(); err != nil {
		return nil, fmt.Errorf("Dify 配置验证失败: %w", err)
	}

	// 创建 Dify 客户端
	difyClient := dify.NewDifyClient(&ds.config.Dify)

	// 创建处理选项
	processingOptions := dify.DefaultProcessingOptions()
	processingOptions.DefaultRemindBefore = ds.config.Reminder.DefaultRemindBefore

	// 创建处理器
	difyProcessor := dify.NewProcessor(difyClient, "dify-service-user", processingOptions)

	// 处理图片 - 提供文件名参数
	fileName := fmt.Sprintf("clipboard_%s.png", time.Now().Format("20060102_150405"))
	processingResp, err := difyProcessor.ProcessImage(ctx, imageData, fileName)
	if err != nil {
		return nil, err
	}

	// 转换为 models.DifyResponse
	return ds.convertProcessingResponseToDifyResponse(processingResp), nil
}

// convertProcessingResponseToDifyResponse 将ProcessingResponse转换为DifyResponse
func (ds *DifyServiceImpl) convertProcessingResponseToDifyResponse(resp *dify.ProcessingResponse) *models.DifyResponse {
	difyResp := &models.DifyResponse{
		CreatedAt: resp.Timestamp,
		Metadata:  make(map[string]interface{}),
	}

	// 将处理结果添加到Answer字段
	if resp.Success && resp.Reminder != nil {
		// 成功情况，将提醒信息序列化为JSON字符串
		if reminderJSON, err := json.Marshal(resp.Reminder); err == nil {
			difyResp.Answer = string(reminderJSON)
		}
	} else {
		// 失败情况，将错误信息放入Answer
		difyResp.Answer = resp.ErrorMessage
	}

	// 添加元数据
	difyResp.Metadata["success"] = resp.Success
	difyResp.Metadata["processing_time"] = resp.ProcessingTime.String()
	difyResp.Metadata["request_id"] = resp.RequestID

	if resp.ParsedInfo != nil {
		difyResp.Metadata["confidence"] = resp.ParsedInfo.Confidence
	}

	return difyResp
}

// ValidateConfig 验证配置
func (ds *DifyServiceImpl) ValidateConfig() error {
	if ds.config == nil {
		return fmt.Errorf("配置未初始化")
	}

	return ds.config.Dify.Validate()
}

// TestConnection 测试 Dify 服务连接
func (ds *DifyServiceImpl) TestConnection() error {
	// 1. 配置验证
	if err := ds.ValidateConfig(); err != nil {
		return err
	}

	// 2. 创建客户端并发送 HTTP HEAD 请求检查API端点可达性
	difyClient := dify.NewDifyClient(&ds.config.Dify)

	// 使用客户端的验证方法测试连接
	if err := difyClient.ValidateConfig(); err != nil {
		return fmt.Errorf("Dify API 连接测试失败: %w", err)
	}

	return nil
}