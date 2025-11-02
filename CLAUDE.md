# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application called `to_icalendar` that sends reminders to iOS Reminders via the Pushcut service. It allows users to create JSON-formatted reminders and send them to iOS devices where they are automatically converted to iOS Reminders through Shortcuts.

## Architecture

The application follows a clean modular structure with the following main components:

### Core Modules
- **main.go**: CLI entry point handling commands (init, upload, test)
- **internal/pushcut**: Pushcut API client for communicating with Pushcut service
- **internal/config**: Configuration and reminder file management
- **internal/models**: Data structures for reminders and configuration

### Key Data Flow
1. User creates JSON reminder files
2. ConfigManager loads and validates reminders
3. PushcutClient sends reminder data to Pushcut API
4. Pushcut service triggers iOS Shortcuts
5. iOS Shortcuts create reminders in the iOS Reminders app

## Common Development Commands

### Building
```bash
go mod tidy
go build -o to_icalendar main.go
```

### Running the Application
```bash
# Initialize configuration files
./to_icalendar init

# Send single reminder
./to_icalendar upload config/reminder.json

# Batch send reminders
./to_icalendar upload reminders/*.json

# Test Pushcut connection
./to_icalendar test
```

### Development Setup
The application requires no external test framework or build tools beyond the standard Go toolchain. Configuration files are created automatically by the `init` command.

## Configuration Structure

### Server Configuration (config/server.yaml)
```yaml
pushcut:
  api_key: "your_pushcut_api_key"
  webhook_id: "your_webhook_id"
  timezone: "Asia/Shanghai"
```

### Reminder Format (config/reminder.json)
```json
{
  "title": "Meeting Reminder",
  "description": "Attend product review meeting",
  "date": "2024-12-25",
  "time": "14:30",
  "remind_before": "15m",
  "priority": "medium",
  "list": "Work"
}
```

## Pushcut Integration

The application communicates with Pushcut's API using:
- HTTP POST requests with JSON payload
- Bearer token authentication with API keys
- Webhook endpoints for triggering iOS Shortcuts
- Structured data format for iOS Shortcuts consumption

## Error Handling Patterns

- All functions return explicit errors for debugging
- Validation occurs at multiple stages (file format, time logic, API connectivity)
- Graceful degradation when optional features fail
- Clear error messages with context for troubleshooting
- HTTP response status checking and error logging

## Time Zone Handling

- Time zone configuration in server.yaml
- All internal processing preserves local timezone context
- Proper timezone conversion for reminder time calculation
- Duration parsing for reminder alarms (supports m/h/d suffixes)

## iOS Shortcuts Integration

The application sends data in a format compatible with iOS Shortcuts:
- Structured JSON with all reminder fields
- Title, description, date, time, priority, and list information
- Remind-before duration for alarm settings
- Cross-platform compatibility for any device that can make HTTP requests

## Security Considerations

- API keys stored in plain text configuration files
- No sensitive data encryption required (compared to previous CalDAV approach)
- HTTPS communication with Pushcut API endpoints
- No password storage or management needed