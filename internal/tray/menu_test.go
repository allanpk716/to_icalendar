package tray

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMenuItem(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		label        string
		tooltip      string
		menuType     MenuType
		action       string
		shortcut     string
		isSeparator  bool
		isEnabled    bool
		order        int
		expectedType MenuType
	}{
		{
			name:         "创建动作菜单项",
			id:           "exit",
			label:        "退出",
			tooltip:      "退出应用程序",
			menuType:     MenuTypeAction,
			action:       "quit",
			shortcut:     "Ctrl+Q",
			isSeparator:  false,
			isEnabled:    true,
			order:        1,
			expectedType: MenuTypeAction,
		},
		{
			name:         "创建分隔符菜单项",
			id:           "",
			label:        "",
			tooltip:      "",
			menuType:     MenuTypeSeparator,
			action:       "",
			shortcut:     "",
			isSeparator:  true,
			isEnabled:    true,
			order:        2,
			expectedType: MenuTypeSeparator,
		},
		{
			name:         "创建子菜单项",
			id:           "settings",
			label:        "设置",
			tooltip:      "打开设置",
			menuType:     MenuTypeSubmenu,
			action:       "open_settings",
			shortcut:     "",
			isSeparator:  false,
			isEnabled:    true,
			order:        3,
			expectedType: MenuTypeSubmenu,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := NewMenuItem(tt.id, tt.label, tt.tooltip, tt.menuType, tt.action, tt.shortcut, tt.isSeparator, tt.isEnabled, tt.order)

			assert.Equal(t, tt.expectedType, item.Type)
			assert.Equal(t, tt.label, item.Label)
			assert.Equal(t, tt.tooltip, item.Tooltip)
			assert.Equal(t, tt.action, item.Action)
			assert.Equal(t, tt.shortcut, item.Shortcut)
			assert.Equal(t, tt.isSeparator, item.IsSeparator)
			assert.Equal(t, tt.isEnabled, item.IsEnabled)
			assert.Equal(t, tt.order, item.Order)

			// 如果没有提供ID，应该自动生成
			if tt.id == "" {
				assert.NotEmpty(t, item.ID)
			} else {
				assert.Equal(t, tt.id, item.ID)
			}
		})
	}
}

func TestNewSeparatorMenuItem(t *testing.T) {
	separator := NewSeparatorMenuItem(1)

	assert.NotEmpty(t, separator.ID)
	assert.True(t, separator.IsSeparator)
	assert.Equal(t, MenuTypeSeparator, separator.Type)
	assert.True(t, separator.IsEnabled)
	assert.Equal(t, 1, separator.Order)
	assert.Empty(t, separator.Label)
	assert.Empty(t, separator.Action)
}

func TestMenuItem_Validate(t *testing.T) {
	tests := []struct {
		name        string
		item        *MenuItem
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效动作菜单项",
			item: &MenuItem{
				ID:     "exit",
				Label:  "退出",
				Type:   MenuTypeAction,
				Action: "quit",
				Order:  1,
			},
			expectError: false,
		},
		{
			name: "有效分隔符",
			item: &MenuItem{
				ID:          "sep1",
				Type:        MenuTypeSeparator,
				IsSeparator: true,
				Order:       1,
			},
			expectError: false,
		},
		{
			name: "空标签的非分隔符应该失败",
			item: &MenuItem{
				ID:    "empty",
				Type:  MenuTypeAction,
				Order: 1,
			},
			expectError: true,
			errorMsg:    "菜单项标签不能为空",
		},
		{
			name: "负数顺序应该失败",
			item: &MenuItem{
				ID:    "invalid",
				Label: "Invalid",
				Type:  MenuTypeAction,
				Order: -1,
			},
			expectError: true,
			errorMsg:    "菜单项顺序不能为负数",
		},
		{
			name: "动作类型没有动作应该失败",
			item: &MenuItem{
				ID:    "no-action",
				Label: "No Action",
				Type:  MenuTypeAction,
				Order: 1,
			},
			expectError: true,
			errorMsg:    "动作类型菜单项必须指定动作",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMenuItem_SetEnabled(t *testing.T) {
	item := NewMenuItem("test", "Test", "Test tooltip", MenuTypeAction, "test_action", "", false, true, 1)

	// 默认应该是启用状态
	assert.True(t, item.IsEnabled)

	// 禁用菜单项
	item.SetEnabled(false)
	assert.False(t, item.IsEnabled)

	// 重新启用菜单项
	item.SetEnabled(true)
	assert.True(t, item.IsEnabled)
}

func TestMenuItem_SetLabel(t *testing.T) {
	item := NewMenuItem("test", "Test", "Test tooltip", MenuTypeAction, "test_action", "", false, true, 1)

	// 设置新标签
	newLabel := "New Test Label"
	item.SetLabel(newLabel)

	assert.Equal(t, newLabel, item.Label)
}

func TestNewTrayMenu(t *testing.T) {
	menu := NewTrayMenu()

	require.NotNil(t, menu)
	assert.NotEmpty(t, menu.ID)
	assert.Empty(t, menu.AppID)
	assert.Empty(t, menu.Items)
	assert.True(t, menu.IsActive)
	assert.NotZero(t, menu.CreatedAt)
	assert.NotZero(t, menu.UpdatedAt)
}

func TestTrayMenu_AddItem(t *testing.T) {
	menu := NewTrayMenu()
	item := NewMenuItem("exit", "退出", "退出应用程序", MenuTypeAction, "quit", "", false, true, 1)

	// 添加菜单项
	menu.AddItem(item)

	assert.Equal(t, 1, len(menu.Items))
	assert.Equal(t, item.ID, menu.Items[0].ID)
	assert.Equal(t, item.Label, menu.Items[0].Label)
}

func TestTrayMenu_AddItem_Nil(t *testing.T) {
	menu := NewTrayMenu()

	// 尝试添加nil项
	menu.AddItem(nil)

	assert.Equal(t, 0, len(menu.Items))
}

func TestTrayMenu_RemoveItem(t *testing.T) {
	menu := NewTrayMenu()
	item1 := NewMenuItem("exit", "退出", "退出应用程序", MenuTypeAction, "quit", "", false, true, 1)
	item2 := NewMenuItem("settings", "设置", "打开设置", MenuTypeAction, "open_settings", "", false, true, 2)

	// 添加菜单项
	menu.AddItem(item1)
	menu.AddItem(item2)

	assert.Equal(t, 2, len(menu.Items))

	// 移除菜单项
	removed := menu.RemoveItem("exit")
	assert.True(t, removed)
	assert.Equal(t, 1, len(menu.Items))
	assert.Equal(t, item2.ID, menu.Items[0].ID)

	// 尝试移除不存在的菜单项
	removedAgain := menu.RemoveItem("nonexistent")
	assert.False(t, removedAgain)
	assert.Equal(t, 1, len(menu.Items))
}

func TestTrayMenu_GetItem(t *testing.T) {
	menu := NewTrayMenu()
	item1 := NewMenuItem("exit", "退出", "退出应用程序", MenuTypeAction, "quit", "", false, true, 1)
	item2 := NewMenuItem("settings", "设置", "打开设置", MenuTypeAction, "open_settings", "", false, true, 2)

	// 添加菜单项
	menu.AddItem(item1)
	menu.AddItem(item2)

	// 获取存在的菜单项
	foundItem, exists := menu.GetItem("exit")
	assert.True(t, exists)
	assert.Equal(t, item1.ID, foundItem.ID)

	// 获取不存在的菜单项
	_, exists = menu.GetItem("nonexistent")
	assert.False(t, exists)
}

func TestTrayMenu_SortItems(t *testing.T) {
	menu := NewTrayMenu()

	// 添加乱序的菜单项
	item3 := NewMenuItem("item3", "Item 3", "", MenuTypeAction, "action3", "", false, true, 3)
	item1 := NewMenuItem("item1", "Item 1", "", MenuTypeAction, "action1", "", false, true, 1)
	item2 := NewMenuItem("item2", "Item 2", "", MenuTypeAction, "action2", "", false, true, 2)

	menu.AddItem(item3)
	menu.AddItem(item1)
	menu.AddItem(item2)

	// 验证初始顺序是乱的
	assert.Equal(t, item3.ID, menu.Items[0].ID)
	assert.Equal(t, item1.ID, menu.Items[1].ID)
	assert.Equal(t, item2.ID, menu.Items[2].ID)

	// 排序菜单项
	menu.SortItems()

	// 验证排序后的顺序
	assert.Equal(t, item1.ID, menu.Items[0].ID)
	assert.Equal(t, item2.ID, menu.Items[1].ID)
	assert.Equal(t, item3.ID, menu.Items[2].ID)
}

func TestTrayMenu_Validate(t *testing.T) {
	tests := []struct {
		name        string
		menu        *TrayMenu
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效菜单",
			menu: &TrayMenu{
				ID: "valid-menu",
				Items: []MenuItem{
					{
						ID:    "exit",
						Label: "退出",
						Type:  MenuTypeAction,
						Order: 1,
					},
				},
			},
			expectError: false,
		},
		{
			name: "空菜单应该失败",
			menu: &TrayMenu{
				ID:    "empty-menu",
				Items: []MenuItem{},
			},
			expectError: true,
			errorMsg:    "托盘菜单不能为空",
		},
		{
			name: "包含无效菜单项应该失败",
			menu: &TrayMenu{
				ID: "invalid-menu",
				Items: []MenuItem{
					{
						ID:    "invalid",
						Type:  MenuTypeAction,
						Order: -1, // 无效顺序
					},
				},
			},
			expectError: true,
			errorMsg:    "菜单项 invalid 验证失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.menu.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTrayMenu_SetAppID(t *testing.T) {
	menu := NewTrayMenu()
	appID := "test-app-id"

	menu.SetAppID(appID)
	assert.Equal(t, appID, menu.AppID)
}

func TestTrayMenu_SetActive(t *testing.T) {
	menu := NewTrayMenu()

	// 默认应该是激活状态
	assert.True(t, menu.IsActive)

	// 设置为非激活
	menu.SetActive(false)
	assert.False(t, menu.IsActive)

	// 重新激活
	menu.SetActive(true)
	assert.True(t, menu.IsActive)
}

func TestTrayMenu_Clear(t *testing.T) {
	menu := NewTrayMenu()

	// 添加一些菜单项
	item1 := NewMenuItem("item1", "Item 1", "", MenuTypeAction, "action1", "", false, true, 1)
	item2 := NewMenuItem("item2", "Item 2", "", MenuTypeAction, "action2", "", false, true, 2)

	menu.AddItem(item1)
	menu.AddItem(item2)

	assert.Equal(t, 2, len(menu.Items))

	// 清空菜单
	menu.Clear()

	assert.Equal(t, 0, len(menu.Items))
}

func TestTrayMenu_GetActionItems(t *testing.T) {
	menu := NewTrayMenu()

	// 添加不同类型的菜单项
	actionItem1 := NewMenuItem("action1", "Action 1", "", MenuTypeAction, "do_something", "", false, true, 1)
	actionItem2 := NewMenuItem("action2", "Action 2", "", MenuTypeAction, "do_else", "", false, true, 2)
	separator := NewSeparatorMenuItem(3)
	submenuItem := NewMenuItem("submenu1", "Submenu 1", "", MenuTypeSubmenu, "", "", false, true, 4)

	menu.AddItem(actionItem1)
	menu.AddItem(separator)
	menu.AddItem(submenuItem)
	menu.AddItem(actionItem2)

	// 获取动作类型的菜单项
	actionItems := menu.GetActionItems()

	assert.Equal(t, 2, len(actionItems))
	assert.Contains(t, []string{actionItems[0].ID, actionItems[1].ID}, actionItem1.ID)
	assert.Contains(t, []string{actionItems[0].ID, actionItems[1].ID}, actionItem2.ID)
}

func TestTrayMenu_GetEnabledItems(t *testing.T) {
	menu := NewTrayMenu()

	// 添加启用和禁用的菜单项
	enabledItem := NewMenuItem("enabled", "Enabled", "", MenuTypeAction, "do_enabled", "", false, true, 1)
	disabledItem := NewMenuItem("disabled", "Disabled", "", MenuTypeAction, "do_disabled", "", false, false, 2)

	menu.AddItem(enabledItem)
	menu.AddItem(disabledItem)

	// 获取启用的菜单项
	enabledItems := menu.GetEnabledItems()

	assert.Equal(t, 1, len(enabledItems))
	assert.Equal(t, enabledItem.ID, enabledItems[0].ID)
}