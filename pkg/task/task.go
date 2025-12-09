package task

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/models"
)

// TaskSession 单次clip-upload任务的完整会话记录
type TaskSession struct {
	TaskID       string                 `json:"task_id"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time,omitempty"`
	Status       TaskStatus             `json:"status"`
	ImageHash    string                 `json:"image_hash,omitempty"`
	TaskDir      string                 `json:"task_dir"`
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description,omitempty"`
	DifySuccess  bool                   `json:"dify_success"`
	TodoSuccess  bool                   `json:"todo_success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Files        map[string]TaskFile    `json:"files"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
)

// TaskFile 任务相关文件信息
type TaskFile struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Type      FileType  `json:"type"`
}

// FileType 文件类型
type FileType string

const (
	FileTypeClipboardOriginal FileType = "clipboard_original"
	FileTypeClipboardProcessed FileType = "clipboard_processed"
	FileTypeDifyRequest      FileType = "dify_request"
	FileTypeDifyResponse     FileType = "dify_response"
	FileTypeTaskInfo         FileType = "task_info"
	FileTypeTodoResult       FileType = "todo_result"
)

// TaskIndex 任务索引记录
type TaskIndex struct {
	TaskID      string    `json:"task_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Status      TaskStatus `json:"status"`
	ImageHash   string    `json:"image_hash,omitempty"`
	Title       string    `json:"title,omitempty"`
	TaskDir     string    `json:"task_dir"`
	Size        int64     `json:"size"`        // 任务目录总大小
	FileCount   int       `json:"file_count"`  // 文件数量
	DifySuccess bool      `json:"dify_success"`
	TodoSuccess bool      `json:"todo_success"`
}


// TaskManager 任务管理器
type TaskManager struct {
	baseDir       string
	tasksDir      string
	globalDir     string
	indexFile     string
	mutex         sync.RWMutex
	sessions      map[string]*TaskSession
	index         []*TaskIndex
	cacheConfig   models.CacheConfig
	logger        *log.Logger
}

// NewTaskManager 创建新的任务管理器
func NewTaskManager(baseDir string, cacheConfig models.CacheConfig, logger *log.Logger) (*TaskManager, error) {
	if logger == nil {
		logger = log.Default()
	}

	tm := &TaskManager{
		baseDir:     baseDir,
		tasksDir:    filepath.Join(baseDir, "cache", "tasks"),
		globalDir:   filepath.Join(baseDir, "cache", "global"),
		indexFile:   filepath.Join(baseDir, "cache", "global", "task_index.json"),
		sessions:    make(map[string]*TaskSession),
		index:       make([]*TaskIndex, 0),
		cacheConfig: cacheConfig,
		logger:      logger,
	}

	// 创建必要的目录
	if err := tm.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 加载任务索引
	if err := tm.loadIndex(); err != nil {
		tm.logf("加载任务索引失败: %v", err)
	}

	return tm, nil
}

// generateTaskID 生成唯一的任务ID
func (tm *TaskManager) generateTaskID() string {
	timestamp := time.Now().Format("2006-01-02_150405")

	// 使用当前时间的微秒和进程ID生成哈希
	data := fmt.Sprintf("%s_%d", timestamp, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	shortHash := hex.EncodeToString(hash[:])[:8]

	return fmt.Sprintf("%s_%s", timestamp, shortHash)
}

// CreateTaskSession 创建新的任务会话
func (tm *TaskManager) CreateTaskSession() (*TaskSession, error) {
	taskID := tm.generateTaskID()
	taskDir := filepath.Join(tm.tasksDir, taskID)

	// 创建任务目录
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return nil, fmt.Errorf("创建任务目录失败: %w", err)
	}

	session := &TaskSession{
		TaskID:    taskID,
		StartTime: time.Now(),
		Status:    TaskStatusRunning,
		TaskDir:   taskDir,
		Files:     make(map[string]TaskFile),
		Metadata:  make(map[string]interface{}),
	}

	// 保存任务信息到文件
	if err := tm.saveTaskInfo(session); err != nil {
		tm.logf("保存任务信息失败: %v", err)
	}

	// 添加到内存中的会话
	tm.mutex.Lock()
	tm.sessions[taskID] = session
	tm.mutex.Unlock()

	tm.logf("创建任务会话: %s", taskID)
	return session, nil
}

// SaveFileToTask 保存文件到任务目录
func (tm *TaskManager) SaveFileToTask(session *TaskSession, fileType FileType, data []byte, filename ...string) error {
	var fileName string
	if len(filename) > 0 && filename[0] != "" {
		fileName = filename[0]
	} else {
		fileName = tm.getDefaultFileName(fileType)
	}

	filePath := filepath.Join(session.TaskDir, fileName)

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	// 记录文件信息
	taskFile := TaskFile{
		Name:      fileName,
		Path:      filePath,
		Size:      int64(len(data)),
		CreatedAt: time.Now(),
		Type:      fileType,
	}

	session.Files[string(fileType)] = taskFile
	session.Metadata["last_updated"] = time.Now()

	// 更新任务信息文件
	if err := tm.saveTaskInfo(session); err != nil {
		tm.logf("更新任务信息失败: %v", err)
	}

	tm.logf("保存文件到任务 %s: %s (%d bytes)", session.TaskID, fileName, len(data))
	return nil
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(session *TaskSession, status TaskStatus, errorMsg ...string) {
	session.Status = status
	session.EndTime = time.Now()

	if len(errorMsg) > 0 {
		session.ErrorMessage = errorMsg[0]
	}

	// 更新任务信息文件
	if err := tm.saveTaskInfo(session); err != nil {
		tm.logf("更新任务信息失败: %v", err)
	}

	// 更新索引
	if err := tm.updateIndex(session); err != nil {
		tm.logf("更新任务索引失败: %v", err)
	}

	tm.logf("任务 %s 状态更新为: %s", session.TaskID, status)
}

// SetImageHash 设置图片哈希
func (tm *TaskManager) SetImageHash(session *TaskSession, imageHash string) {
	session.ImageHash = imageHash
	session.Metadata["image_hash_set"] = time.Now()

	if err := tm.saveTaskInfo(session); err != nil {
		tm.logf("设置图片哈希失败: %v", err)
	}
}

// SetReminderInfo 设置提醒信息
func (tm *TaskManager) SetReminderInfo(session *TaskSession, reminder *models.ParsedReminder) {
	session.Title = reminder.Original.Title
	session.Description = reminder.Original.Description
	session.Metadata["list"] = reminder.List
	session.Metadata["date"] = reminder.Original.Date
	session.Metadata["time"] = reminder.Original.Time

	if err := tm.saveTaskInfo(session); err != nil {
		tm.logf("设置提醒信息失败: %v", err)
	}
}

// SetDifyResult 设置Dify处理结果
func (tm *TaskManager) SetDifyResult(session *TaskSession, success bool, requestData, responseData []byte) error {
	session.DifySuccess = success
	session.Metadata["dify_processed"] = time.Now()

	// 保存请求和响应数据
	if len(requestData) > 0 {
		if err := tm.SaveFileToTask(session, FileTypeDifyRequest, requestData, "dify_request.json"); err != nil {
			return err
		}
	}

	if len(responseData) > 0 {
		if err := tm.SaveFileToTask(session, FileTypeDifyResponse, responseData, "dify_response.json"); err != nil {
			return err
		}
	}

	return nil
}

// SetTodoResult 设置Todo处理结果
func (tm *TaskManager) SetTodoResult(session *TaskSession, success bool, todoData []byte) error {
	session.TodoSuccess = success
	session.Metadata["todo_processed"] = time.Now()

	if len(todoData) > 0 {
		if err := tm.SaveFileToTask(session, FileTypeTodoResult, todoData, "todo_result.json"); err != nil {
			return err
		}
	}

	return nil
}

// GetTaskSession 获取任务会话
func (tm *TaskManager) GetTaskSession(taskID string) (*TaskSession, error) {
	tm.mutex.RLock()
	if session, exists := tm.sessions[taskID]; exists {
		tm.mutex.RUnlock()
		return session, nil
	}
	tm.mutex.RUnlock()

	// 如果内存中没有，尝试从文件加载
	return tm.loadTaskSession(taskID)
}

// GetRecentTasks 获取最近的任务列表
func (tm *TaskManager) GetRecentTasks(limit int) ([]*TaskIndex, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if limit <= 0 || limit > len(tm.index) {
		limit = len(tm.index)
	}

	// 返回最近的任务（按时间倒序）
	result := make([]*TaskIndex, limit)
	copy(result, tm.index[:limit])

	return result, nil
}

// 以下为私有方法

// ensureDirectories 确保目录结构存在
func (tm *TaskManager) ensureDirectories() error {
	dirs := []string{tm.tasksDir, tm.globalDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}

	return nil
}

// getDefaultFileName 获取默认文件名
func (tm *TaskManager) getDefaultFileName(fileType FileType) string {
	switch fileType {
	case FileTypeClipboardOriginal:
		return "clipboard_original.png"
	case FileTypeClipboardProcessed:
		return "clipboard_processed.png"
	case FileTypeDifyRequest:
		return "dify_request.json"
	case FileTypeDifyResponse:
		return "dify_response.json"
	case FileTypeTaskInfo:
		return "task_info.json"
	case FileTypeTodoResult:
		return "todo_result.json"
	default:
		return "unknown_file"
	}
}

// saveTaskInfo 保存任务信息到文件
func (tm *TaskManager) saveTaskInfo(session *TaskSession) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化任务信息失败: %w", err)
	}

	filePath := filepath.Join(session.TaskDir, "task_info.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入任务信息文件失败: %w", err)
	}

	return nil
}

// loadTaskSession 从文件加载任务会话
func (tm *TaskManager) loadTaskSession(taskID string) (*TaskSession, error) {
	taskDir := filepath.Join(tm.tasksDir, taskID)
	infoFile := filepath.Join(taskDir, "task_info.json")

	data, err := os.ReadFile(infoFile)
	if err != nil {
		return nil, fmt.Errorf("读取任务信息文件失败: %w", err)
	}

	var session TaskSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("解析任务信息文件失败: %w", err)
	}

	// 添加到内存缓存
	tm.mutex.Lock()
	tm.sessions[taskID] = &session
	tm.mutex.Unlock()

	return &session, nil
}

// loadIndex 加载任务索引
func (tm *TaskManager) loadIndex() error {
	if _, err := os.Stat(tm.indexFile); os.IsNotExist(err) {
		return nil // 文件不存在，返回空索引
	}

	data, err := os.ReadFile(tm.indexFile)
	if err != nil {
		return fmt.Errorf("读取任务索引文件失败: %w", err)
	}

	if err := json.Unmarshal(data, &tm.index); err != nil {
		return fmt.Errorf("解析任务索引文件失败: %w", err)
	}

	tm.logf("加载了 %d 个任务索引记录", len(tm.index))
	return nil
}

// updateIndex 更新任务索引
func (tm *TaskManager) updateIndex(session *TaskSession) error {
	// 计算任务目录大小和文件数量
	var totalSize int64
	var fileCount int

	for _, file := range session.Files {
		totalSize += file.Size
		fileCount++
	}

	// 创建索引记录
	index := &TaskIndex{
		TaskID:      session.TaskID,
		StartTime:   session.StartTime,
		EndTime:     session.EndTime,
		Status:      session.Status,
		ImageHash:   session.ImageHash,
		Title:       session.Title,
		TaskDir:     session.TaskDir,
		Size:        totalSize,
		FileCount:   fileCount,
		DifySuccess: session.DifySuccess,
		TodoSuccess: session.TodoSuccess,
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 查找是否已存在该任务的索引
	found := false
	for i, existing := range tm.index {
		if existing.TaskID == session.TaskID {
			tm.index[i] = index
			found = true
			break
		}
	}

	// 如果不存在，添加到开头（最新的在前）
	if !found {
		tm.index = append([]*TaskIndex{index}, tm.index...)
	}

	// 异步保存索引文件
	go func() {
		if err := tm.saveIndex(); err != nil {
			tm.logf("保存任务索引失败: %v", err)
		}
	}()

	return nil
}

// saveIndex 保存任务索引到文件
func (tm *TaskManager) saveIndex() error {
	data, err := json.MarshalIndent(tm.index, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化任务索引失败: %w", err)
	}

	// 原子写入
	tempFile := tm.indexFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时索引文件失败: %w", err)
	}

	if err := os.Rename(tempFile, tm.indexFile); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("重命名索引文件失败: %w", err)
	}

	return nil
}

// logf 记录日志
func (tm *TaskManager) logf(format string, args ...interface{}) {
	if tm.logger != nil {
		tm.logger.Printf("[TaskManager] "+format, args...)
	} else {
		fmt.Printf("[TaskManager] %s\n", fmt.Sprintf(format, args...))
	}
}