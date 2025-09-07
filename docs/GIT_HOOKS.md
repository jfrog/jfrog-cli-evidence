# Git Hooks Setup Guide

**Author**: yevdoa

## Overview

This project uses Git hooks to ensure code quality and consistency before commits and pushes. The hooks automatically run various checks defined in the Makefile to catch issues early in the development process.

## Available Hooks

### Pre-commit Hook
Runs before each commit to ensure basic code quality:
- **Code Formatting** (`make fmt`) - Formats Go code using gofmt
- **Code Vetting** (`make vet`) - Runs go vet to catch suspicious constructs
- **Short Tests** (`make test-short`) - Runs quick unit tests

### Pre-push Hook
Runs before pushing to remote repository with comprehensive checks:
- **Code Formatting** (`make fmt`) - Ensures consistent code formatting
- **Code Vetting** (`make vet`) - Static analysis for potential issues
- **Linting** (`make lint`) - Runs golangci-lint for code quality
- **Full Test Suite** (`make test`) - Runs all tests including integration tests

## Installation

### Quick Install
The easiest way to install the git hooks is using the Makefile:

```bash
make install-hooks
```

### Manual Install
Alternatively, you can run the setup script directly:

```bash
./scripts/setup-git-hooks.sh
```

### What Gets Installed
The installation process will:
1. Create executable hook files in `.git/hooks/`
2. Back up any existing hooks (with timestamp)
3. Set proper permissions for the hook scripts

## Usage

Once installed, the hooks will run automatically:

- **On commit**: The pre-commit hook runs automatically when you use `git commit`
- **On push**: The pre-push hook runs automatically when you use `git push`

### Skipping Hooks (Emergency Use Only)

If you need to bypass the hooks temporarily (not recommended):

```bash
# Skip pre-commit hook
git commit --no-verify -m "Your message"

# Skip pre-push hook
git push --no-verify
```

⚠️ **Warning**: Only skip hooks in emergency situations. The hooks are there to maintain code quality.

## Uninstallation

To remove the git hooks:

```bash
make uninstall-hooks
```

Or manually:

```bash
rm .git/hooks/pre-commit
rm .git/hooks/pre-push
```

## Troubleshooting

### Hook Fails but Code Seems Fine

1. **Check formatting**: Run `make fmt` to auto-fix formatting issues
2. **Check linter warnings**: Run `make lint` to see detailed linting issues
3. **Check test failures**: Run `make test` to see which tests are failing

### Hook Takes Too Long

The pre-commit hook runs only short tests for speed. If it's still slow:
- Consider optimizing your short tests
- Ensure tests marked as integration tests use the appropriate build tag

### Permission Denied Error

If you get a permission error when the hook runs:

```bash
chmod +x .git/hooks/pre-commit
chmod +x .git/hooks/pre-push
```

### Hook Not Running

Verify the hooks are installed:

```bash
ls -la .git/hooks/ | grep -E "(pre-commit|pre-push)"
```

If not present, run `make install-hooks` again.

## Customization

### Modifying Hook Behavior

The hook behaviors are defined in the Makefile targets:
- Edit the `pre-commit` target in `Makefile` to change pre-commit checks
- Edit the `pre-push` target in `Makefile` to change pre-push checks

After modifying the Makefile, reinstall the hooks:

```bash
make install-hooks
```

### Adding New Hooks

To add additional git hooks:

1. Add a new target in the Makefile
2. Update `scripts/setup-git-hooks.sh` to include the new hook
3. Run `make install-hooks`

## Best Practices

1. **Don't skip hooks regularly** - They're there to help maintain quality
2. **Fix issues immediately** - Don't let formatting or linting issues accumulate
3. **Keep tests fast** - Especially those run in pre-commit
4. **Update hooks together** - When changing hook behavior, ensure all team members update

## Integration with CI/CD

The same checks run by the git hooks should also run in your CI/CD pipeline:
- This ensures consistency between local and remote checks
- Prevents issues if someone skips hooks locally
- The Makefile targets make this integration straightforward

## Team Collaboration

When working in a team:

1. **Document hook requirements** in your README
2. **Include hook installation** in your onboarding process
3. **Keep hooks lightweight** to avoid developer frustration
4. **Communicate changes** when updating hook behavior

## Support

If you encounter issues with the git hooks:
1. Check this documentation
2. Review the Makefile targets
3. Examine the hook scripts in `.git/hooks/`
4. Check the setup script at `scripts/setup-git-hooks.sh`
