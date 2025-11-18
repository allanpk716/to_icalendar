<!--
Sync Impact Report:
Version change: N/A → 1.0.0
List of modified principles:
  - Template placeholders → Code Quality Excellence
  - Template placeholders → Test-First Development
  - Template placeholders → User Experience Consistency
  - Template placeholders → Performance & Reliability Standards
Added sections:
  - Code Quality Excellence
  - Test-First Development
  - User Experience Consistency
  - Performance & Reliability Standards
  - Development Workflow
  - Quality Gates
Removed sections: None
Templates requiring updates:
  ✅ .specify/templates/plan-template.md - Constitution Check aligned
  ✅ .specify/templates/spec-template.md - Testing requirements aligned
  ✅ .specify/templates/tasks-template.md - Task categories reflect new principles
Follow-up TODOs: None
-->

# to_icalendar Constitution

## Core Principles

### I. 代码质量卓越 (Code Quality Excellence)

每个代码变更都必须维护或提升代码库的整体质量。所有代码必须清晰、可维护且经过适当审查。

- **代码可读性优先**：变量名、函数名和注释必须使用中文，与现有代码库保持语言一致
- **SOLID原则强制执行**：所有新组件必须遵循单一职责、开闭原则、里氏替换、接口隔离和依赖倒置
- **DRY原则**：杜绝重复代码，通过抽象和重构提高代码复用性
- **KISS简洁设计**：追求最简单可行的解决方案，避免过度工程化
- **错误处理规范**：使用显式错误返回模式，提供充分上下文信息，使用github.com/WQGroup/logger记录结构化日志

### II. 测试优先开发 (Test-First Development)

测试驱动开发不是可选项，而是强制要求。功能实现前必须先编写失败的测试用例。

- **TDD强制流程**：编写测试 → 确认测试失败 → 实现功能 → 测试通过 → 重构优化
- **测试覆盖率要求**：所有新功能必须有单元测试，核心业务逻辑必须有集成测试
- **测试分类明确**：单元测试针对单个函数，集成测试验证模块交互，端到端测试验证完整用户场景
- **使用testify框架**：统一使用github.com/stretchr/testify进行测试断言和模拟
- **测试即文档**：测试用例必须作为功能使用说明，展示预期行为和边界条件

### III. 用户体验一致性 (User Experience Consistency)

所有用户交互必须保持一致性和直观性，确保跨平台体验的统一标准。

- **CLI接口标准**：所有命令必须支持help标志，错误信息输出到stderr，正常结果输出到stdout
- **错误信息本地化**：所有面向用户的错误和提示信息必须使用中文，提供清晰的解决建议
- **配置文件格式**：YAML格式用于配置，JSON格式用于数据，保持现有文件结构和命名约定
- **跨平台兼容**：确保在Windows、Linux、macOS上行为一致，处理路径分隔符差异
- **交互一致性**：相同类型的操作必须在所有命令中表现一致（如文件选择、确认提示）

### IV. 性能与可靠性标准 (Performance & Reliability Standards)

应用程序必须在不同负载条件下保持稳定性能，并提供可靠的错误恢复机制。

- **响应时间要求**：Microsoft Graph API调用必须在30秒内完成，本地配置操作必须在1秒内完成
- **内存使用优化**：批量处理大量提醒事项时，内存使用不能超过可用内存的50%
- **错误恢复能力**：网络失败必须实现重试机制（最多3次），配置文件损坏必须提供恢复建议
- **并发安全**：所有并发操作必须使用适当的同步机制，避免竞态条件
- **资源清理**：所有网络连接、文件句柄和其他资源必须在操作结束时正确释放

## Development Workflow

### 代码审查流程

所有代码变更必须经过同行评审，确保符合代码质量标准和架构原则。

- **审查清单强制**：必须检查代码可读性、测试覆盖、错误处理、性能影响和安全性
- **小步提交**：每个PR应该专注于单一功能或修复，便于审查和回滚
- **自动化检查**：代码格式、静态分析和测试必须通过才能合并
- **文档同步更新**：代码变更必须同时更新相关文档和注释

### 版本管理策略

遵循语义化版本控制，确保向后兼容性和清晰的变更沟通。

- **主版本号**：不兼容的API变更
- **次版本号**：向后兼容的功能新增
- **修订号**：向后兼容的问题修正
- **分支策略**：主分支保护，功能分支开发，标签标记版本

## Quality Gates

### 发布前检查清单

每个发布必须满足以下质量标准，确保用户获得稳定可靠的体验。

- **测试覆盖率**：新功能测试覆盖率不低于80%，核心逻辑100%
- **性能基准**：关键操作性能不能劣于上一版本
- **兼容性验证**：在支持的操作系统上进行功能验证
- **文档完整性**：README、配置示例和故障排除指南必须更新
- **安全审查**：依赖项安全扫描，凭据处理安全检查

### 持续集成要求

自动化流水线必须确保每次代码提交都符合质量标准。

- **静态代码分析**：Go vet、gofmt和golint检查必须通过
- **依赖项检查**：自动检测过时依赖和安全漏洞
- **构建验证**：确保代码在所有目标平台上成功编译
- **自动化测试**：单元测试和集成测试必须100%通过

## Governance

### 宪法至高无上

本宪法超越所有其他开发实践和流程规范。任何冲突都必须以宪法为准。

- **合宪性审查**：所有PR和设计决策必须验证是否符合宪法原则
- **宪法修订流程**：修订需要文档说明、团队同意和迁移计划
- **合规监督**：指定技术负责人负责监督宪法执行
- **争议解决**：开发中的争议通过宪法原则进行裁决

- **版本**: 1.0.0 | **批准日期**: 2024-11-18 | **最后修订**: 2024-11-18