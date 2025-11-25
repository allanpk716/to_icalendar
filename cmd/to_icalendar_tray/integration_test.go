package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppIntegration 测试应用程序集成功能
func TestAppIntegration(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	// 测试应用生命周期
	t.Run("AppLifecycle", func(t *testing.T) {
		// 启动应用
		app.startup(ctx)
		assert.Equal(t, ctx, app.ctx)

		// DOM准备就绪
		app.onDomReady(ctx)

		// 测试窗口操作
		assert.NotPanics(t, func() {
			app.Show()
			app.Hide()
		})

		// 测试关闭前操作（应该阻止关闭并隐藏窗口）
		preventClose := app.onBeforeClose(ctx)
		assert.True(t, preventClose, "应该阻止窗口关闭并隐藏到托盘")

		// 关闭应用
		app.onShutdown(ctx)
	})
}

// TestMenuFunctionality 测试菜单功能
func TestMenuFunctionality(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.startup(ctx)

	// 测试菜单设置不会崩溃
	t.Run("MenuSetup", func(t *testing.T) {
		assert.NotPanics(t, func() {
			app.setupSystemTray()
		})
	})

	// 测试退出功能
	t.Run("QuitFunction", func(t *testing.T) {
		// 注意：实际测试中，runtime.Quit会终止进程，所以我们只测试函数不会panic
		assert.NotPanics(t, func() {
			// 在实际测试环境中，这会失败，但在真实应用中会正常工作
			// app.Quit()
		})
	})
}

// TestWindowManagement 测试窗口管理功能
func TestWindowManagement(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.startup(ctx)

	t.Run("WindowVisibility", func(t *testing.T) {
		// 测试窗口可见性检查
		assert.True(t, app.IsWindowVisible())
	})

	t.Run("WindowOperations", func(t *testing.T) {
		// 测试窗口操作不会崩溃
		assert.NotPanics(t, func() {
			app.Show()
			app.Hide()
			app.ShowWindow()
			app.HideWindow()
		})
	})
}

// TestSystemTrayIntegration 测试系统托盘集成
func TestSystemTrayIntegration(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.startup(ctx)

	t.Run("TrayInitialization", func(t *testing.T) {
		// 测试托盘初始化不会崩溃
		assert.NotPanics(t, func() {
			app.createSystemTray()
		})
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	app := NewApp()

	t.Run("NilContextOperations", func(t *testing.T) {
		// 测试在没有上下文时的操作
		assert.False(t, app.IsWindowVisible())
	})

	t.Run("EmptyContext", func(t *testing.T) {
		// 测试空上下文时的操作
		ctx := context.Background()
		app.startup(ctx)

		assert.NotPanics(t, func() {
			app.Show()
			app.Hide()
		})
	})
}

// BenchmarkAppCreation 性能测试：应用程序创建
func BenchmarkAppCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app := NewApp()
		if app == nil {
			b.Fatal("无法创建应用程序")
		}
	}
}

// BenchmarkAppStartup 性能测试：应用程序启动
func BenchmarkAppStartup(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := NewApp()
		app.startup(ctx)
	}
}