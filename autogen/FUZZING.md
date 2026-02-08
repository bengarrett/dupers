# Fuzz Testing for Dupers

This document describes the fuzz testing implementation for the dupers project.

## Overview

Fuzz testing (or fuzzing) is an automated software testing technique that involves providing invalid, unexpected, or random data as inputs to a computer program. The goal is to find coding errors and security vulnerabilities.

## Fuzz Test Implementation

We've implemented fuzz tests for the most critical components of dupers:

### 1. File Parsing and Checksum Calculation (`pkg/dupe/parse`)

- **FuzzChecksum**: Tests the SHA-256 checksum calculation with various file contents
  - Ensures no panics occur with different file contents
  - Verifies deterministic output (same input → same checksum)
  - Handles edge cases like empty files, binary data, large files

### 2. Archive Handling (`pkg/dupe/internal/archive`)

- **FuzzExtension**: Tests file extension detection
  - Handles various filename formats and edge cases
  - Validates extension format (must start with '.')
  - Tests case sensitivity and special characters

- **FuzzMIME**: Tests MIME type detection by filename
  - Validates MIME type lookup logic
  - Tests various archive formats and extensions

- **FuzzReadMIME**: Tests MIME type reading from actual files
  - Handles file I/O errors gracefully
  - Validates MIME type detection from file contents

### 3. Command Processing (`pkg/cmd`)

- **FuzzWindowsChk**: Tests Windows directory path validation
  - Handles various path formats (quoted, unquoted, escaped)
  - Validates Windows-specific path issues
  - Ensures graceful error handling

- **FuzzSearchSummary**: Tests search result summary formatting
  - Validates output formatting with different inputs
  - Handles edge cases in result counts and search terms

## Running Fuzz Tests

### Basic Fuzz Testing

Run all fuzz tests with a 30-second timeout:

```bash
task fuzz
```

### Continuous Fuzzing

Run fuzz tests continuously until interrupted (useful for long-running fuzz sessions):

```bash
task fuzz-all
```

### Individual Fuzz Tests

Run specific fuzz tests with custom timing:

```bash
# Run checksum fuzzing for 60 seconds
go test -fuzz=FuzzChecksum -fuzztime=60s ./pkg/dupe/parse

# Run extension fuzzing for 120 seconds
go test -fuzz=FuzzExtension -fuzztime=120s ./pkg/dupe/internal/archive

# Run Windows path validation fuzzing
go test -fuzz=FuzzWindowsChk -fuzztime=30s ./pkg/cmd
```

### Fuzz Test Discovery

To see all available fuzz tests:

```bash
go test -fuzztarget=./... 2>/dev/null | grep "Fuzz"
```

## Fuzz Test Corpus

Each fuzz test includes a set of initial test cases (corpus) that provide good coverage of common scenarios:

- **Checksum**: Empty files, small files, large files, binary data, repeated patterns
- **Extension**: Common extensions, no extensions, hidden files, complex filenames
- **MIME**: Various archive formats, case sensitivity, compound extensions
- **WindowsChk**: Quoted paths, escaped paths, various Windows path formats
- **SearchSummary**: Different result counts, special characters, Unicode

## Fuzz Test Results

When a fuzz test finds an issue, it will:

1. **Fail the test** and provide details about the input that caused the failure
2. **Save the failing input** to the test corpus for reproduction
3. **Provide a stack trace** showing where the failure occurred

Example of a fuzz test failure:

```
--- FAIL: FuzzChecksum (0.01s)
    --- FAIL: FuzzChecksum (0.00s)
        panic: runtime error: slice bounds out of range [recovered]
            panic: runtime error: slice bounds out of range

    Failing input written to testdata/fuzz/FuzzChecksum/... 
    To re-run:
    go test -run=FuzzChecksum/...
```

## Integration with CI/CD

Fuzz tests can be integrated into your CI/CD pipeline:

```yaml
# Example GitHub Actions workflow
- name: Fuzz Testing
  run: task fuzz
```

For continuous fuzzing, consider using dedicated fuzzing services or longer-running workflows.

## Best Practices

1. **Run fuzz tests regularly**: Integrate them into your development workflow
2. **Monitor for failures**: Fuzz tests can uncover rare edge cases
3. **Update corpus**: Add interesting test cases to the initial corpus
4. **Balance coverage**: Focus on critical paths and security-sensitive code
5. **Monitor performance**: Fuzz tests should run efficiently

## Troubleshooting

### Import Cycles

If you encounter import cycles when adding new fuzz tests:
- Ensure fuzz test files don't create circular dependencies
- Use simple helper functions within the fuzz test file
- Avoid importing packages that might create cycles

### Slow Fuzz Tests

If fuzz tests run too slowly:
- Reduce the `-fuzztime` parameter
- Focus on specific fuzz targets
- Optimize the fuzz test logic
- Run fewer iterations in CI/CD

### Fuzz Test Failures

If a fuzz test fails:
1. Examine the failing input
2. Reproduce the issue with the saved corpus
3. Fix the underlying code issue
4. Add the failing case to the initial corpus
5. Verify the fix with `go test -run=FuzzTarget`

## Resources

- [Go Fuzzing Documentation](https://go.dev/security/fuzz/)
- [Go Fuzzing Tutorial](https://go.dev/doc/tutorial/fuzz)
- [Fuzzing Best Practices](https://github.com/golang/go/wiki/Fuzzing)

## Contributing

When adding new features or modifying existing code:

1. **Identify fuzz-worthy functions**: Look for functions that process untrusted input
2. **Add appropriate fuzz tests**: Cover edge cases and security-sensitive paths
3. **Update the corpus**: Add meaningful initial test cases
4. **Run existing fuzz tests**: Ensure no regressions
5. **Document new fuzz targets**: Update this documentation

Fuzz testing is an essential part of our quality assurance process and helps ensure the reliability and security of the dupers application.