# Test Reporting Documentation

## Overview

This repository uses comprehensive GitHub Actions workflows to provide detailed test reporting, coverage analysis, and quality metrics. The test reporting system makes it easy to identify failed tests, track coverage trends, and maintain code quality.

## Features

### 🎯 Core Features

1. **Detailed Test Results**
   - JUnit XML reports for test visualization
   - JSON output for programmatic analysis
   - Pass/fail/skip statistics per test run
   - Test duration tracking

2. **Coverage Analysis**
   - Line-by-line code coverage reports
   - HTML coverage visualization
   - XML coverage for CI integration
   - Package-level coverage breakdown
   - Coverage trend tracking

3. **PR Integration**
   - Automatic PR comments with test summaries
   - Status checks for test results
   - Coverage badge updates
   - Failed test details in PR comments

4. **Multi-Platform Testing**
   - Test matrix across different OS (Ubuntu, macOS, Windows)
   - Multiple Go version support
   - Parallel test execution

## Workflows

### 1. Main Test Workflow (`test.yml`)

The primary testing workflow that runs on every push and pull request.

**Features:**
- Runs unit tests with race detection
- Generates multiple report formats (JUnit, JSON, HTML)
- Creates GitHub Step Summary with results
- Comments on PRs with detailed statistics
- Uploads artifacts for further analysis

**Triggers:**
- Push to any branch
- Pull requests
- Manual workflow dispatch

### 2. Test Reporter (`test-reporter.yml`)

Post-processing workflow that analyzes test results after the main test run.

**Features:**
- Downloads and parses test artifacts
- Creates detailed test reports
- Updates PR status checks
- Generates coverage badges

### 3. Coverage Badge (`coverage-badge.yml`)

Maintains up-to-date coverage badges for the repository.

**Features:**
- Runs on main branch pushes
- Calculates coverage percentage
- Updates README badge automatically
- Integrates with Codecov

### 4. Test Matrix (`test-matrix.yml`)

Comprehensive testing across multiple environments.

**Features:**
- Tests on Ubuntu, macOS, and Windows
- Multiple Go versions (1.22, 1.23, 1.24)
- Parallel execution for faster results
- Combined summary report

### 5. Test Dashboard (`test-dashboard.yml`)

Interactive HTML dashboard for test results visualization.

**Features:**
- Historical test trend charts
- Coverage progression graphs
- Package-level coverage details
- Deployable to GitHub Pages

## Makefile Targets

Enhanced Makefile targets for local testing:

```bash
# Run all tests with coverage
make test

# Run unit tests only
make test-unit

# Generate JUnit XML reports
make test-junit

# Generate JSON test output
make test-json

# Generate HTML coverage report
make coverage

# Generate XML coverage report
make coverage-xml
```

## Test Output Formats

### JUnit XML
- **Location:** `test-results/junit.xml`
- **Use:** GitHub Actions test reporting, CI/CD integration
- **Viewers:** GitHub Actions, Jenkins, CircleCI

### JSON Output
- **Location:** `test-results/test.json`
- **Use:** Programmatic analysis, custom reporting
- **Format:** Go test2json format

### Coverage Reports
- **Text:** `coverage/coverage-summary.txt` - Quick overview
- **HTML:** `coverage/coverage.html` - Interactive browser view
- **XML:** `coverage/coverage.xml` - SonarCloud/Codecov integration
- **JSON:** `coverage/coverage.json` - Custom analysis

## PR Comment Example

When tests run on a pull request, an automatic comment is added:

```markdown
## ✅ Test Results for abc1234

### 📊 Summary
- **Status:** All tests passed!
- **Coverage:** ![Coverage](https://img.shields.io/badge/coverage-78.5%25-green)
- **Duration:** [View in Actions](link)

### 📈 Test Statistics
| Metric | Count | Percentage |
|--------|-------|------------|
| Total | 156 | 100% |
| ✅ Passed | 156 | 100.0% |
| ❌ Failed | 0 | 0.0% |
| ⏭️ Skipped | 0 | 0.0% |
```

## Configuration

### SonarCloud Integration

Configure in `sonar-project.properties`:
```properties
sonar.projectKey=jfrog-cli-evidence
sonar.go.coverage.reportPaths=coverage/coverage.out
sonar.test.inclusions=**/*_test.go
```

### Codecov Integration

Add `CODECOV_TOKEN` to repository secrets for automatic coverage upload.

## Best Practices

1. **Write Comprehensive Tests**
   - Aim for >80% code coverage
   - Include unit and integration tests
   - Test edge cases and error conditions

2. **Review Test Reports**
   - Check PR comments for test results
   - Review coverage reports for gaps
   - Monitor test duration trends

3. **Fix Failing Tests Immediately**
   - Tests must pass before merging
   - Investigate flaky tests
   - Keep test suite fast and reliable

4. **Use Test Artifacts**
   - Download test results for debugging
   - Analyze JSON output for patterns
   - Share HTML coverage reports

## Troubleshooting

### Common Issues

1. **Tests Pass Locally but Fail in CI**
   - Check for race conditions (`-race` flag)
   - Verify environment variables
   - Review OS-specific code

2. **Coverage Not Updating**
   - Ensure tests generate `coverage.out`
   - Check file paths in workflows
   - Verify tool installations

3. **PR Comments Not Appearing**
   - Check workflow permissions
   - Verify GitHub token availability
   - Review action logs for errors

### Debug Commands

```bash
# Run tests with verbose output
go test -v ./evidence/...

# Generate coverage profile
go test -coverprofile=coverage.out ./evidence/...

# View coverage in terminal
go tool cover -func=coverage.out

# Open HTML coverage in browser
go tool cover -html=coverage.out
```

## Integration with IDEs

### VS Code
- Install "Go" extension
- Run tests from sidebar
- View inline coverage highlights

### GoLand/IntelliJ
- Built-in test runner
- Coverage visualization
- Test history tracking

## Future Enhancements

- [ ] Benchmark result tracking
- [ ] Performance regression detection
- [ ] Test flakiness detection
- [ ] Custom test badges
- [ ] Slack/Discord notifications
- [ ] Test result APIs

## Support

For issues or questions about test reporting:
1. Check workflow logs in GitHub Actions
2. Review this documentation
3. Open an issue with test output attached
