package tray

import (
	"fmt"
	"time"
)

// MenuType 菜单项类型
type MenuType string

const (
	MenuTypeAction    MenuType = "action"    // 执行动作
	MenuTypeSubmenu   MenuType = "submenu"   // 子菜单
	MenuTypeSeparator MenuType = "separator" // 分隔符
)

// MenuItem 托盘菜单项定义
type MenuItem struct {
	ID          string      `json:"id"`
	Label       string      `json:"label"`
	Tooltip     string      `json:"tooltip,omitempty"`
	Type        MenuType    `json:"type"`
	Action      string      `json:"action,omitempty"`
	Shortcut    string      `json:"shortcut,omitempty"`
	IsSeparator bool        `json:"is_separator"`
	IsEnabled   bool        `json:"is_enabled"`
	Order       int         `json:"order"`
}

// NewMenuItem 创建新的菜单项
func NewMenuItem(id, label, tooltip string, menuType MenuType, action, shortcut string, isSeparator, isEnabled bool, order int) *MenuItem {
	if id == "" {
		id = generateID()
	}

	return &MenuItem{
		ID:          id,
		Label:       label,
		Tooltip:     tooltip,
		Type:        menuType,
		Action:      action,
		Shortcut:    shortcut,
		IsSeparator: isSeparator,
		IsEnabled:   isEnabled,
		Order:       order,
	}
}

// NewSeparatorMenuItem 创建分隔符菜单项
func NewSeparatorMenuItem(order int) *MenuItem {
	return &MenuItem{
		ID:          generateID(),
		Type:        MenuTypeSeparator,
		IsSeparator: true,
		IsEnabled:   true,
		Order:       order,
	}
}

// Validate 验证菜单项配置
func (mi *MenuItem) Validate() error {
	if mi.Label == "" && !mi.IsSeparator {
		return fmt.Errorf("菜单项标签不能为空")
	}
	if mi.Order < 0 {
		return fmt.Errorf("菜单项顺序不能为负数")
	}
	if mi.Type == MenuTypeAction && mi.Action == "" && !mi.IsSeparator {
		return fmt.Errorf("动作类型菜单项必须指定动作")
	}
	return nil
}

// SetEnabled 设置是否启用
func (mi *MenuItem) SetEnabled(enabled bool) {
	mi.IsEnabled = enabled
}

// SetLabel 设置标签
func (mi *MenuItem) SetLabel(label string) {
	mi.Label = label
}

// TrayMenu 托盘右键菜单配置
type TrayMenu struct {
	ID       string      `json:"id"`
	AppID    string      `json:"app_id"`
	Items    []MenuItem  `json:"items"`
	IsActive bool        `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// NewTrayMenu 创建新的托盘菜单
func NewTrayMenu() *TrayMenu {
	now := time.Now()
	return &TrayMenu{
		ID:        generateID(),
		AppID:     "",
		Items:     make([]MenuItem, 0),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddItem 添加菜单项
func (tm *TrayMenu) AddItem(item *MenuItem) {
	if item != nil {
		tm.Items = append(tm.Items, *item)
		tm.UpdatedAt = time.Now()
	}
}

// RemoveItem 移除菜单项
func (tm *TrayMenu) RemoveItem(id string) bool {
	for i, item := range tm.Items {
		if item.ID == id {
			tm.Items = append(tm.Items[:i], tm.Items[i+1:]...)
			tm.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetItem 获取菜单项
func (tm *TrayMenu) GetItem(id string) (*MenuItem, bool) {
	for i, item := range tm.Items {
		if item.ID == id {
			return &tm.Items[i], true
		}
	}
	return nil, false
}

// SortItems 按Order字段排序菜单项
func (tm *TrayMenu) SortItems() {
	// 简单的冒泡排序
	for i := 0; i < len(tm.Items)-1; i++ {
		for j := 0; j < len(tm.Items)-i-1; j++ {
			if tm.Items[j].Order > tm.Items[j+1].Order {
				tm.Items[j], tm.Items[j+1] = tm.Items[j+1], tm.Items[j]
			}
		}
	}
	tm.UpdatedAt = time.Now()
}

// Validate 验证托盘菜单配置
func (tm *TrayMenu) Validate() error {
	if len(tm.Items) == 0 {
		return fmt.Errorf("托盘菜单不能为空")
	}

	for _, item := range tm.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("菜单项 %s 验证失败: %w", item.ID, err)
		}
	}

	return nil
}

// SetAppID 设置关联的应用程序ID
func (tm *TrayMenu) SetAppID(appID string) {
	tm.AppID = appID
	tm.UpdatedAt = time.Now()
}

// SetActive 设置是否激活
func (tm *TrayMenu) SetActive(active bool) {
	tm.IsActive = active
	tm.UpdatedAt = time.Now()
}

// Clear 清空所有菜单项
func (tm *TrayMenu) Clear() {
	tm.Items = make([]MenuItem, 0)
	tm.UpdatedAt = time.Now()
}

// GetActionItems 获取所有动作类型的菜单项
func (tm *TrayMenu) GetActionItems() []MenuItem {
	var actionItems []MenuItem
	for _, item := range tm.Items {
		if item.Type == MenuTypeAction {
			actionItems = append(actionItems, item)
		}
	}
	return actionItems
}

// GetEnabledItems 获取所有启用的菜单项
func (tm *TrayMenu) GetEnabledItems() []MenuItem {
	var enabledItems []MenuItem
	for _, item := range tm.Items {
		if item.IsEnabled {
			enabledItems = append(enabledItems, item)
		}
	}
	return enabledItems
}