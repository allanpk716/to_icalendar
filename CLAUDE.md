# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application called `to_icalendar` that sends reminders to Microsoft Todo. It allows users to create JSON-formatted reminders and send them to Microsoft Todo via the Microsoft Graph API.

## Architecture

The application follows a clean modular structure with the following main components:

### Core Modules
- **main.go**: CLI entry point handling commands (init, upload, test)
- **internal/microsoft-todo**: Microsoft Todo API client for communicating with Microsoft Graph service
- **internal/config**: Configuration and reminder file management
- **internal/models**: Data structures for reminders and configuration

### Key Data Flow
1. User creates JSON reminder files
2. ConfigManager loads and validates reminders
3. Microsoft Todo client sends reminder data to Microsoft Graph API
4. Microsoft Graph API creates tasks in Microsoft Todo

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

# Test Microsoft Todo connection
./to_icalendar test
```

### Development Setup
The application requires no external test framework or build tools beyond the standard Go toolchain. Configuration files are created automatically by the `init` command.

## Configuration Structure

### Server Configuration (config/server.yaml)
```yaml
microsoft_todo:
  tenant_id: "YOUR_TENANT_ID"
  client_id: "YOUR_CLIENT_ID"
  client_secret: "YOUR_CLIENT_SECRET"
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

## Microsoft Todo Integration

The application communicates with Microsoft Graph API using:
- OAuth 2.0 authentication with Azure AD
- HTTP POST requests with JSON payload
- Structured data format for Microsoft Todo task creation
- Tasks.ReadWrite.All API permissions for task management

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

## Microsoft Todo Integration Details

The application sends data in a format compatible with Microsoft Todo:
- Structured JSON with all reminder fields
- Title, description, date, time, priority, and list information
- Remind-before duration for alarm settings
- Cross-platform compatibility for any device that can access Microsoft Todo

## Security Considerations

- Azure AD credentials stored in plain text configuration files
- OAuth 2.0 authentication with secure token management
- HTTPS communication with Microsoft Graph API endpoints
- No password storage or management needed
- Requires Azure AD application registration and API permissions
- 使用 github.com/WQGroup/logger 作为项目的日志库
- 使用 github.com/WQGroup/logger 库来记录日志