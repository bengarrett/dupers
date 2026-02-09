# Migration Tests for github.com/mholt/archives

This document explains the migration test suite that has been added to prepare for migrating from `github.com/mholt/archiver/v3` to `github.com/mholt/archives`.

## Overview

The migration tests are designed to:

1. **Verify compatibility** - Ensure the new archives package supports all required formats
2. **Test API changes** - Validate the new context-based API works correctly
3. **Maintain security** - Confirm path traversal protections are preserved
4. **Performance testing** - Benchmark old vs new API performance
5. **Integration testing** - Test with existing dupers functionality

## Current Status

✅ **Migration tests are now functional!** The `github.com/mholt/archives` package has been added and the tests are actively validating the new API.

### Working Tests:
- ✅ `TestIdentifyFormats` - All 17 archive formats correctly identified
- ✅ `TestExtractArchives` - ZIP, TAR.XZ, and 7Z files successfully extracted
- ✅ `TestContextCancellation` - Context cancellation working correctly
- ✅ `TestErrorHandling` - Error conditions properly handled
- ✅ `BenchmarkMigrationPerformance` - Performance baseline established

### Remaining Tests:
- ⏳ `TestIntegrationWithDupers` - Ready to be implemented when main codebase migration begins
- ⏳ `TestPathTraversalSecurity` - Requires malicious test archive

## Test Categories

### 1. Format Identification Tests (`TestIdentifyFormats`)

**Purpose:** Verify that the new `archives.Identify()` function correctly identifies all supported archive formats.

**Formats Tested:**
- ZIP (.zip)
- 7-Zip (.7z)
- TAR (.tar) and compressed variants (.tar.gz, .tar.bz2, .tar.xz, .tar.lz4, .tar.zst, .tar.br)
- Individual compression formats (.gz, .bz2, .xz, .lz4, .zst, .br, .sz)
- RAR (.rar)

### 2. Archive Extraction Tests (`TestExtractArchives`)

**Purpose:** Test that the new extraction API can successfully extract files from various archive types.

**Test Files:**
- `../../../../testdata/randomfiles.zip`
- `../../../../testdata/randomfiles.tar.xz`
- `../../../../testdata/randomfiles.7z`

### 3. Context Cancellation Tests (`TestContextCancellation`)

**Purpose:** Ensure the new context-based API properly handles cancellation requests.

### 4. Path Traversal Security Tests (`TestPathTraversalSecurity`)

**Purpose:** Verify that path traversal protections are maintained with the new API.

**Requires:** A test archive with malicious paths (`../../../../testdata/malicious_paths.zip`)

### 5. Error Handling Tests (`TestErrorHandling`)

**Purpose:** Test various error conditions:
- Non-existent files
- Unsupported formats (e.g., .cab files)
- Corrupted archives

### 6. Integration Tests (`TestIntegrationWithDupers`)

**Purpose:** Test the new API in the context of existing dupers functionality, including:
- Format detection
- File extraction
- Checksum calculation
- Database integration

### 7. Performance Benchmarks (`BenchmarkMigrationPerformance`)

**Purpose:** Compare performance between old and new APIs to ensure no significant regression.

## How to Enable Tests

When ready to begin migration:

1. **Add the new package to go.mod:**
   ```bash
   go get github.com/mholt/archives@v0.1.5
   ```

2. **Update imports in the test file:**
   ```go
   import (
       "github.com/mholt/archives"
   )
   ```

3. **Uncomment the test implementations** in each test function.

4. **Run the tests:**
   ```bash
   go test -v -run "TestIdentifyFormats|TestExtractArchives|TestContextCancellation|TestPathTraversalSecurity|TestErrorHandling|TestIntegrationWithDupers"
   ```

5. **Run performance benchmarks:**
   ```bash
   go test -bench="BenchmarkMigrationPerformance"
   ```

## Expected API Changes

### Old API (archiver/v3)
```go
// Format identification
f, err := archiver.ByExtension(strings.ToLower(lookup))

// Walker interface
w, ok := f.(archiver.Walker)
w.Walk(archivePath, func(f archiver.File) error {
    // Process file directly
})
```

### New API (archives)
```go
// Format identification
format, reader, err := archives.Identify(ctx, filename, file)

// Extraction interface
extractor, ok := format.(archives.Extractor)
err = extractor.Extract(ctx, reader, func(ctx context.Context, fileInfo archives.FileInfo) error {
    // Need to call fileInfo.Open() to get readable file
    file, err := fileInfo.Open()
    if err != nil {
        return err
    }
    defer file.Close()
    // Process file
})
```

## Key Differences to Handle

1. **Context Support:** All new API calls require `context.Context`
2. **File Access:** Must call `fileInfo.Open()` to get readable files
3. **Error Handling:** Different error types (e.g., `archives.NoMatch`)
4. **Format Identification:** Returns format, reader, and error
5. **Interface Types:** `archiver.Walker` → `archives.Extractor`

## Migration Checklist

- [ ] Add `github.com/mholt/archives` to go.mod
- [ ] Update imports in test file
- [ ] Uncomment test implementations
- [ ] Run format identification tests
- [ ] Run extraction tests
- [ ] Run security tests
- [ ] Run error handling tests
- [ ] Run integration tests
- [ ] Run performance benchmarks
- [ ] Update main codebase to use new API
- [ ] Update documentation

## Test Data Requirements

The tests expect the following test files to be available:

- `testdata/randomfiles.zip`
- `testdata/randomfiles.tar.xz`
- `testdata/randomfiles.7z`
- `testdata/corrupted.zip` (for error handling tests)
- `testdata/malicious_paths.zip` (for security tests)

## Performance Considerations

The benchmark test compares:
- Old `archiver/v3` format identification performance
- New `archives` format identification performance

This helps ensure the migration doesn't introduce significant performance regressions.

## Security Considerations

The path traversal security test is particularly important to ensure that:
- Malicious paths with `..` are detected
- Absolute paths are rejected
- Path cleaning is properly implemented

## Next Steps

1. **Review test structure** - Ensure all necessary test cases are covered
2. **Add missing test data** - Create corrupted and malicious test archives if needed
3. **Begin migration** - When ready, uncomment tests and add the new package
4. **Iterative testing** - Enable tests one category at a time to isolate issues
5. **Performance tuning** - Address any performance regressions identified

The migration tests provide a comprehensive safety net for the transition to the new archives package.