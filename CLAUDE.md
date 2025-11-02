# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application called `to_icalendar` that sends reminders to iOS Reminders via the CalDAV protocol. It allows users to create JSON-formatted reminders and upload them directly to Apple's iCloud calendar service.

## Architecture

The application follows a clean modular structure with the following main components:

### Core Modules
- **main.go**: CLI entry point handling commands (init, upload, test, list)
- **internal/caldav**: CalDAV client for communicating with iCloud servers
- **internal/ical**: iCalendar VTODO component creation and formatting
- **internal/config**: Configuration and reminder file management
- **internal/crypto**: AES-256-GCM encryption for password storage
- **internal/models**: Data structures for reminders and configuration

### Key Data Flow
1. User creates JSON reminder files
2. ConfigManager loads and validates reminders
3. ICalCreator converts reminders to iCalendar VTODO format
4. CalDAVClient uploads to iCloud using HTTP requests
5. PasswordManager securely handles Apple ID app-specific passwords

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

# Upload single reminder
./to_icalendar upload config/reminder.json

# Batch upload reminders
./to_icalendar upload reminders/*.json

# Test CalDAV connection
./to_icalendar test

# List existing reminders
./to_icalendar list
```

### Development Setup
The application requires no external test framework or build tools beyond the standard Go toolchain. Configuration files are created automatically by the `init` command.

## Configuration Structure

### Server Configuration (config/server.yaml)
```yaml
caldav:
  server_url: "https://caldav.icloud.com"
  username: "your_apple_id@icloud.com"
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

## Security Implementation

- Passwords are encrypted using AES-256-GCM
- Encryption keys are derived from machine hardware characteristics
- Encrypted passwords stored in `data/.encrypted_password` with 0600 permissions
- Sensitive data in memory is cleared after use

## CalDAV Integration

The application communicates with iCloud's CalDAV service using:
- HTTP PUT requests for uploading VTODO components
- PROPFIND requests for listing and testing connections
- Basic authentication with Apple ID and app-specific passwords
- Custom filename generation based on timestamp and reminder title

## Error Handling Patterns

- All functions return explicit errors for debugging
- Validation occurs at multiple stages (file format, time logic, CalDAV connectivity)
- Graceful degradation when optional features fail
- Clear error messages with context for troubleshooting

## Time Zone Handling

- Time zone configuration in server.yaml
- All internal processing in UTC
- Proper timezone conversion for user-facing dates
- Duration parsing for reminder alarms (supports m/h/d suffixes)