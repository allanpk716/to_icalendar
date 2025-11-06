# Test Data

This directory contains test fixtures and sample data used by the test suite.

## Files

### Configuration Files
- `config_valid.yaml` - Valid configuration for testing successful scenarios
- `config_invalid.yaml` - Invalid configuration for testing error handling

### Sample Data
- `sample_text.txt` - Sample text inputs for testing text processing
- `sample_reminder.json` - Sample reminder in JSON format

## Usage

These files are used by the unit and integration tests in:
- `../unit/` - Unit tests for individual modules
- `../integration/` - Integration tests for cross-module functionality

## Notes

- All API keys and credentials in these files are for testing purposes only
- Do not use these configurations in production environments
- Files are automatically loaded and cleaned up by the test framework