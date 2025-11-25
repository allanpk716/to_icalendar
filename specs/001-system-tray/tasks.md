---

description: "Task list for System Tray Background Running feature implementation"
---

# Tasks: System Tray Background Running

**Input**: Design documents from `/specs/001-system-tray/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Required per project constitution - minimum 80% unit test coverage with comprehensive integration testing

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Wails desktop app**: `cmd/to_icalendar_tray/backend/`, `cmd/to_icalendar_tray/frontend/`
- **Internal packages**: `internal/tray/`, `internal/models/`
- **Icons**: `cmd/to_icalendar_tray/assets/icons/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create Wails project structure in cmd/to_icalendar_tray/ per implementation plan
- [X] T002 Initialize Wails v2.11.0 project with Vue.js 3.x frontend dependencies
- [X] T003 [P] Create directory structure for backend, frontend, assets, and icons
- [X] T004 Create wails.json configuration file with Windows-specific settings
- [X] T005 [P] Initialize Go module with required dependencies (Wails v2.11.0)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Create tray icon assets directory at cmd/to_icalendar_tray/assets/icons/
- [X] T007 [P] Add placeholder tray icons (16x16, 32x32, 48x48) in PNG format
- [X] T008 Create internal tray package structure in internal/tray/
- [X] T009 Setup base Go structs for tray management in internal/tray/manager.go
- [X] T010 Configure Windows-specific build settings in wails.json
- [X] T011 [P] Setup error handling and logging infrastructure for tray operations

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - 系统托盘图标显示 (Priority: P1) 🎯 MVP

**Goal**: 用户启动to_icalendar应用程序后，程序应该最小化到系统托盘并在系统托盘区域显示应用程序图标，让用户知道程序正在后台运行。

**Independent Test**: 启动应用程序并验证系统托盘区域是否出现应用程序图标，用户能够看到图标并确认程序在后台运行。

### Tests for User Story 1 (Constitution Required - TDD)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T012.1 [P] [US1] Unit test for TrayApplication struct in internal/tray/models_test.go
- [X] T013.1 [P] [US1] Unit test for TrayIcon struct in internal/tray/models_test.go
- [X] T014.1 [P] [US1] Unit test for ApplicationState struct in internal/tray/models_test.go
- [ ] T015.1 [US1] Unit test for tray manager core functionality in internal/tray/manager_test.go
- [ ] T016.1 [US1] Unit test for icon loading functionality in internal/tray/icon_test.go
- [X] T017.1 [US1] Unit test for main app structure in cmd/to_icalendar_tray/app_test.go
- [ ] T018.1 [US1] Integration test for tray initialization in cmd/to_icalendar_tray/integration_test.go
- [ ] T024.1 [US1] End-to-end test for complete tray icon display workflow in tests/e2e/tray_display_test.go

### Implementation for User Story 1

- [X] T012 [P] [US1] Create TrayApplication struct in internal/tray/models.go
- [X] T013 [P] [US1] Create TrayIcon struct in internal/tray/models.go
- [X] T014 [P] [US1] Create ApplicationState struct in internal/tray/models.go
- [X] T015 [US1] Implement core tray manager in internal/tray/manager.go (depends on T012, T013, T014)
- [X] T016 [US1] Create icon loading functionality in internal/tray/icon.go
- [X] T017 [US1] Implement main app structure in cmd/to_icalendar_tray/app.go
- [X] T018 [US1] Create tray initialization logic in cmd/to_icalendar_tray/app.go (depends on T015)
- [X] T019 [US1] Set up Windows-specific tray options in cmd/to_icalendar_tray/app.go
- [X] T020 [US1] Create main application entry point in cmd/to_icalendar_tray/main.go
- [X] T021 [US1] Configure startup behavior to hide window and show tray in cmd/to_icalendar_tray/app.go
- [X] T022 [US1] Add proper shutdown and cleanup logic in cmd/to_icalendar_tray/app.go
- [X] T023 [US1] Create minimal frontend file in cmd/to_icalendar_tray/frontend/src/main.js
- [X] T024 [US1] Test tray icon display and window hiding behavior

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - 右键菜单退出功能 (Priority: P1)

**Goal**: 用户右键点击系统托盘图标时，应该显示一个包含"退出"选项的上下文菜单，用户选择退出后应用程序应该完全关闭。

**Independent Test**: 右键点击托盘图标并选择退出选项，验证程序是否完全关闭且没有残留进程。

### Implementation for User Story 2

- [ ] T025 [P] [US2] Create TrayMenu struct in internal/tray/models.go
- [ ] T026 [P] [US2] Create MenuItem struct in internal/tray/models.go
- [ ] T027 [US2] Implement menu creation logic in internal/tray/menu.go (depends on T025, T026)
- [ ] T028 [US2] Add menu item click handler functionality in internal/tray/menu.go
- [ ] T029 [US2] Integrate menu manager with main tray manager in internal/tray/manager.go
- [ ] T030 [US2] Add exit menu item configuration in cmd/to_icalendar_tray/backend/app.go
- [ ] T031 [US2] Implement clean application shutdown in cmd/to_icalendar_tray/backend/app.go
- [ ] T032 [US2] Add proper resource cleanup on exit in cmd/to_icalendar_tray/backend/app.go

### Tests for User Story 2 (Constitution Required - TDD)

- [ ] T025.1 [P] [US2] Unit test for TrayMenu struct in internal/tray/models_test.go
- [ ] T026.1 [P] [US2] Unit test for MenuItem struct in internal/tray/models_test.go
- [ ] T027.1 [US2] Unit test for menu creation logic in internal/tray/menu_test.go
- [ ] T028.1 [US2] Unit test for menu click handlers in internal/tray/menu_test.go
- [ ] T031.1 [US2] Unit test for clean application shutdown in cmd/to_icalendar_tray/backend/app_test.go
- [ ] T032.1 [US2] Unit test for resource cleanup on exit in cmd/to_icalendar_tray/backend/app_test.go
- [ ] T033.1 [US2] Integration test for right-click menu and exit functionality in tests/integration/menu_exit_test.go

### Implementation for User Story 2

- [ ] T025 [P] [US2] Create TrayMenu struct in internal/tray/models.go
- [ ] T026 [P] [US2] Create MenuItem struct in internal/tray/models.go
- [ ] T027 [US2] Implement menu creation logic in internal/tray/menu.go (depends on T025, T026)
- [ ] T028 [US2] Add menu item click handler functionality in internal/tray/menu.go
- [ ] T029 [US2] Integrate menu manager with main tray manager in internal/tray/manager.go
- [ ] T030 [US2] Add exit menu item configuration in cmd/to_icalendar_tray/backend/app.go
- [ ] T031 [US2] Implement clean application shutdown in cmd/to_icalendar_tray/backend/app.go
- [ ] T032 [US2] Add proper resource cleanup on exit in cmd/to_icalendar_tray/backend/app.go
- [ ] T033 [US2] Manual verification test for right-click menu and exit functionality

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - 后台持续运行 (Priority: P2)

**Goal**: 应用程序最小化到系统托盘后应该继续在后台运行，保持原有的功能和状态，即使主窗口不可见。

**Independent Test**: 最小化应用程序到托盘后验证核心功能是否继续工作，例如检查定时任务是否正常运行。

### Implementation for User Story 3

- [ ] T034 [P] [US3] Create background task manager in internal/tray/tasks.go
- [ ] T035 [US3] Integrate with existing Microsoft Todo functionality in internal/tray/tasks.go
- [ ] T036 [US3] Add resource monitoring for CPU and memory usage in internal/tray/monitor.go
- [ ] T037 [US3] Implement graceful shutdown on system logout/restart in internal/tray/manager.go
- [ ] T038 [US3] Add performance metrics collection in internal/tray/monitor.go
- [ ] T039 [US3] Configure proper signal handling for system events in cmd/to_icalendar_tray/backend/app.go
- [ ] T040 [US3] Test background functionality and resource usage

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T041 [P] Add comprehensive error logging throughout tray components
- [ ] T042 [P] Optimize icon loading and memory usage
- [ ] T043 Add configuration validation and error handling
- [ ] T044 Create documentation for tray functionality
- [ ] T045 Add build scripts for Windows executable generation
- [ ] T046 Test application performance against requirements (≤2s startup, <50MB memory)
- [ ] T046.1 Verify minimum 80% test coverage across all tray components using go test -cover
- [ ] T047 Validate complete functionality works as specified in quickstart.md
- [ ] T048 Implement crash recovery and automatic tray icon cleanup in internal/tray/recovery.go
- [ ] T049 Add Windows-specific signal handling for unexpected termination in cmd/to_icalendar_tray/backend/app.go
- [ ] T050 Test crash scenarios and verify 5-second cleanup window in tests/integration/crash_recovery_test.go

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (US1 → US2 → US3)
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Integrates with US1 but independently testable
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Integrates with US1/US2 but independently testable

### Within Each User Story

- Models before managers
- Core functionality before integration
- Basic implementation before testing
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- Models within each story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all models for User Story 1 together:
Task: "Create TrayApplication struct in internal/tray/models.go"
Task: "Create TrayIcon struct in internal/tray/models.go"
Task: "Create ApplicationState struct in internal/tray/models.go"

# These can run in parallel as they are different files with no dependencies
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready (MVP delivers core tray icon functionality)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (P1 - MVP)
   - Developer B: User Story 2 (P1 - Menu functionality)
   - Developer C: User Story 3 (P2 - Background tasks)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- MVP consists of User Story 1 only (basic tray icon display)
- User Stories 1 & 2 together provide complete basic functionality
- User Story 3 adds advanced background processing capabilities
- Focus on Windows 10+ compatibility as specified
- Performance targets: <3s startup, <50MB memory, <1% idle CPU
- All code should follow Go best practices and project constitution