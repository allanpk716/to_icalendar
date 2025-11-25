package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	app := NewApp()

	require.NotNil(t, app)
	assert.Nil(t, app.ctx) // Context should be nil before startup
}

func TestAppStartup(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	// Test startup
	app.startup(ctx)

	assert.Equal(t, ctx, app.ctx)
}

func TestAppQuit(t *testing.T) {
	app := NewApp()

	// Should not panic
	assert.NotPanics(t, func() {
		app.Quit()
	})
}

func TestAppWindowVisibility(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	// Test with nil context (before startup)
	assert.False(t, app.IsWindowVisible())

	// Test with context (after startup)
	app.startup(ctx)
	assert.True(t, app.IsWindowVisible())
}

func TestAppWindowOperations(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.startup(ctx)

	// Should not panic even though runtime calls will fail in test
	assert.NotPanics(t, func() {
		app.Show()
	})

	assert.NotPanics(t, func() {
		app.Hide()
	})

	assert.NotPanics(t, func() {
		app.ShowWindow()
	})

	assert.NotPanics(t, func() {
		app.HideWindow()
	})
}

func TestAppLifecycle(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	// Test full lifecycle
	assert.NotPanics(t, func() {
		app.startup(ctx)
		app.onDomReady(ctx)
		prevent := app.onBeforeClose(ctx)
		assert.True(t, prevent) // Should prevent close and hide window instead
		app.onShutdown(ctx)
	})
}