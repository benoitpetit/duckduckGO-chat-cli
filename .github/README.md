# 🚀 GitHub Configuration & CI/CD

This directory contains the GitHub Actions workflows, templates, and configuration for the DuckDuckGo Chat CLI project.

## 📁 Contents

- **`workflows/`** - GitHub Actions CI/CD pipelines
- **`pull_request_template.md`** - Template for release PRs

## 🔄 CI/CD Workflows

### 🚀 Release Workflow (`workflows/release.yml`)

**Triggers:**
- PR merged to `prod` branch
- Direct push to `prod` branch  
- Manual trigger with custom version

**Features:**
- Auto-increment version detection
- Cross-platform builds (Linux, Windows, macOS)
- SHA256 checksums generation
- Automatic GitHub release creation
- Release notes generation

### 🧪 Test Workflow (`workflows/test.yml`)

**Triggers:**
- PRs to `master` branch
- Push to `master` branch

**Features:**
- Go code formatting validation
- Static analysis (`go vet`, `golangci-lint`)
- Security scanning (`gosec`)
- Cross-compilation testing
- Dependency verification

## 🎯 Release Process

### 🚀 Method 1: Release Script (Recommended)

```bash
# Interactive mode - guides you through version selection
./scripts/release.sh

# Direct version specification
./scripts/release.sh 1.2.0
```

**What it does:**
1. Validates repository state
2. Creates release branch (`release/vX.X.X`)
3. Pushes to GitHub
4. Provides PR creation instructions

### 📋 Method 2: Manual Process

1. **Create release branch:**
   ```bash
   git checkout -b release/v1.2.0
   git push origin release/v1.2.0
   ```

2. **Create PR:** `release/v1.2.0` → `prod`

3. **Merge PR:** Automatic release is triggered

### ⚡ Method 3: Manual Trigger

GitHub Actions → "Build and Release" → "Run workflow" → Specify version

## 🌳 Branch Strategy

| Branch | Purpose | Triggers |
|--------|---------|----------|
| `master` | Development | Tests on PR/push |
| `prod` | Production | Release on PR merge/push |
| `release/vX.X.X` | Release prep | PR target for `prod` |

## 📦 Release Assets

Each release automatically generates:

### 🗂️ Binaries
- `duckduckgo-chat-cli_vX.X.X_linux_amd64`
- `duckduckgo-chat-cli_vX.X.X_windows_amd64.exe`
- `duckduckgo-chat-cli_vX.X.X_darwin_arm64`

### 🔐 Security
- SHA256 checksums for all binaries
- Release archive (ZIP) with all files

### 📝 Documentation
- Auto-generated release notes
- Installation instructions
- Changelog links

## 🛠️ Development Tools

### Pre-release Validation
```bash
./scripts/pre-release-check.sh
```

**Checks:**
- Go environment and version
- Code formatting (`gofmt`)
- Static analysis (`go vet`)
- Cross-compilation builds
- Git repository state
- Required files presence

### Configuration Files
- **`.golangci.yml`** - Linting rules configuration
- **`pull_request_template.md`** - Standardized PR format

## 🔍 Quality Gates

### Code Quality
- Automated formatting validation
- Static analysis with multiple linters
- Security vulnerability scanning
- Cross-platform build verification

### Release Quality
- Version format validation
- Duplicate version prevention
- Binary functionality testing
- Checksum generation

## 📚 Quick Reference

### Common Commands
```bash
# Check project readiness
./scripts/pre-release-check.sh

# Create new release (interactive)
./scripts/release.sh

# Create specific version
./scripts/release.sh 1.2.0

# Manual build test
./scripts/build.sh
```

### Version Management
- **Patch** (`1.1.5` → `1.1.6`): Bug fixes
- **Minor** (`1.1.5` → `1.2.0`): New features
- **Major** (`1.1.5` → `2.0.0`): Breaking changes

### Workflow URLs
- **Actions:** `https://github.com/benoitpetit/duckduckGO-chat-cli/actions`
- **Releases:** `https://github.com/benoitpetit/duckduckGO-chat-cli/releases`

---

*🤖 This CI/CD setup ensures consistent, automated, and secure releases for the DuckDuckGo Chat CLI project.*
