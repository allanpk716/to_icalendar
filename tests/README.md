# Test Suite

This directory contains the complete test suite for the to_icalendar project, organized according to Go testing best practices.

## Directory Structure

```
tests/
├── unit/                    # Unit tests for individual modules
│   ├── config/             # Configuration management tests
│   ├── clipboard/          # Clipboard functionality tests
│   ├── dify/               # Dify integration tests
│   ├── processors/         # Text/image processing tests
│   └── validators/         # Content validation tests
├── integration/            # Integration tests for cross-module functionality
├── testdata/              # Test fixtures and sample data
├── utils/                 # Test utilities and helpers
├── tools/                 # Manual testing tools
└── README.md              # This file
```

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./tests/unit/...

# Run unit tests for a specific module
go test ./tests/unit/config/...
go test ./tests/unit/processors/...

# Run with verbose output
go test -v ./tests/unit/...

# Run with coverage
go test -cover ./tests/unit/...
```

### Integration Tests
```bash
# Run all integration tests
go test ./tests/integration/...

# Integration tests may require external services
go test -tags=integration ./tests/integration/...
```

### Manual Testing Tools
```bash
# Run manual clipboard testing tool
go run -tags=tools ./tests/tools/manual_clipboard_test.go

# Run integration debugging tool
go run -tags=tools ./tests/tools/debug_integration.go
```

### All Tests
```bash
# Run all tests (unit + integration)
go test ./tests/...

# Run all tests with coverage
go test -cover ./tests/...
```

## Test Types

### Unit Tests (`*_test.go`)
- Test individual functions and methods in isolation
- Fast execution, no external dependencies
- Located in `tests/unit/<module>/`
- Use standard Go testing framework

### Integration Tests
- Test multiple components working together
- May require external services (Dify API, etc.)
- Located in `tests/integration/`
- Use build tags for conditional execution

### Manual Testing Tools (`// +build tools`)
- Standalone executables for manual testing
- Useful for debugging and interactive testing
- Located in `tests/tools/`
- Run with `go run -tags=tools`

## Test Utilities

### Test Environment
The `utils/test_helpers.go` provides:
- `SetupTestEnvironment()` - Creates temporary test environment
- `SetupEmptyTestEnvironment()` - Creates empty test environment
- File and directory management utilities

### Test Fixtures
The `utils/fixtures.go` provides:
- Pre-defined test data for reminders, tasks, and configurations
- Invalid data samples for testing validation
- Sample text inputs for processing tests

## Test Data

### Configuration Files
- `testdata/config_valid.yaml` - Valid configuration
- `testdata/config_invalid.yaml` - Invalid configuration

### Sample Data
- `testdata/sample_text.txt` - Sample text inputs
- `testdata/sample_reminder.json` - Sample reminder JSON

## Best Practices

### Writing Unit Tests
1. Use table-driven tests for multiple scenarios
2. Test both success and failure cases
3. Use descriptive test names
4. Mock external dependencies
5. Use subtests for related scenarios

### Writing Integration Tests
1. Test complete workflows
2. Use temporary directories for file operations
3. Clean up resources after tests
4. Handle optional external dependencies gracefully

### Manual Testing Tools
1. Use build tags to separate from automated tests
2. Provide clear usage instructions
3. Include error handling and user feedback
4. Maintain useful debugging output

## Examples

### Unit Test Example
```go
func TestTextProcessor_QuickAnalyze(t *testing.T) {
    processor, err := processors.NewTextProcessor(nil)
    require.NoError(t, err)

    tests := []struct {
        name  string
        input string
        expected processors.TextAnalysis
    }{
        // test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := processor.QuickAnalyze(tt.input)
            assert.Equal(t, tt.expected.HasDate, result.HasDate)
        })
    }
}
```

### Integration Test Example
```go
func TestProcessingPipeline_FullWorkflow(t *testing.T) {
    env := utils.SetupTestEnvironment(t)
    defer env.Cleanup()

    // Test complete workflow
    result := processText(env, "明天下午2点开会")
    assert.NotNil(t, result)
    env.AssertFileExists(result.JSONPath)
}
```

## Troubleshooting

### Common Issues
1. **Import paths**: Ensure all imports use the correct module path
2. **Build tags**: Use `// +build tools` for manual testing tools
3. **Test data**: Place test data in `testdata/` directory
4. **Temporary files**: Use `t.TempDir()` for temporary test directories

### Running Specific Tests
```bash
# Run a specific test
go test -run TestSpecificFunction ./tests/unit/processors/

# Run tests matching a pattern
go test -run "TestTextProcessor.*" ./tests/unit/processors/

# Skip benchmark tests
go test -short ./tests/unit/...
```

## Contributing

When adding new tests:
1. Place unit tests in the appropriate `tests/unit/<module>/` directory
2. Add integration tests to `tests/integration/`
3. Update fixtures in `utils/fixtures.go` if needed
4. Add test data to `testdata/` directory
5. Update this README if adding new test types

## Coverage

To generate test coverage reports:
```bash
# Generate coverage for unit tests
go test -coverprofile=coverage.out ./tests/unit/...
go tool cover -html=coverage.out

# Generate coverage for all tests
go test -coverprofile=coverage.out ./tests/...
go tool cover -func=coverage.out
```