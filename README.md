# NSHA - Null SHA Fixer

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/RahulGS02/nsha-tool)

A powerful CLI tool to detect and fix null SHA (`0000000...`) and broken tree object issues in Git repositories.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [How It Works](#how-it-works)
- [Project Structure](#project-structure)
- [Technical Specifications](#technical-specifications)
- [Building from Source](#building-from-source)
- [Important Notes](#important-notes)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [License](#license)

## Features

- **Diagnose**: Detect null SHA and broken tree issues in your repository
- **Fix**: Automatically fix issues using git replace --graft and history rewriting
- **Verify**: Verify repository integrity after fixes
- **Fast**: Written in Go for maximum performance
- **Cross-platform**: Works on Windows, Linux, and macOS
- **Beautiful CLI**: Colored output with progress indicators
- **Comprehensive Logging**: Detailed logs and reports saved to user's home directory
- **Safe Backups**: Automatic repository backup before any modifications
- **Dry-Run Mode**: Preview changes before applying them

## Quick Start

### Download Pre-built Binaries

Visit our website: **[https://nsha-tool.netlify.app](https://nsha-tool.netlify.app)**

Or download directly from [GitHub Releases](https://github.com/RahulGS02/nsha-tool/releases/latest)

### Build from Source

```bash
# 1. Build the tool
go build -o nsha

# 2. Diagnose your repository
nsha diagnose

# 3. Fix issues (with dry-run first)
nsha fix --dry-run
nsha fix --yes

# 4. Verify the fix
nsha verify
```

## Installation

### Prerequisites

- **Go 1.21 or higher** - [Download from go.dev](https://go.dev/dl/)
- **Git** - [Download from git-scm.com](https://git-scm.com/downloads)

### Build from Source

#### Windows (PowerShell)
```powershell
# Navigate to project directory
cd c:\path\to\nsha

# Download dependencies
go mod download

# Build the executable
go build -o nsha.exe

# Test it
.\nsha.exe --version
.\nsha.exe --help
```

#### Linux / macOS
```bash
# Navigate to project directory
cd /path/to/nsha

# Download dependencies
go mod download

# Build the executable
go build -o nsha

# Make it executable
chmod +x nsha

# Test it
./nsha --version
./nsha --help

# Optional: Install to system PATH
sudo cp nsha /usr/local/bin/
```

### Build for Multiple Platforms

#### Using PowerShell (Windows)
```powershell
.\build.ps1
```

#### Using Bash (Linux/macOS)
```bash
chmod +x build.sh
./build.sh
```

#### Using Make
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platform
make build-windows
make build-linux
make build-mac
```

This creates executables in the `build/` directory for:
- Windows (amd64, arm64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

## Usage

### Basic Commands

#### 1. Diagnose Issues

Check if your repository has any null SHA or broken tree issues:

```bash
# Diagnose current directory
nsha diagnose

# Diagnose specific repository
nsha diagnose --repo /path/to/repo

# Verbose output
nsha diagnose --verbose
```

**Example Output:**
```
[STEP 1] Diagnosing repository...
[WARNING] Found 3 issue(s):
  1. [null-sha] refs/tags/null-tag: Reference has null SHA
  2. [null-sha] refs/heads/broken-branch: Reference has null SHA
  3. [missing-commit] 0000000000000000000000000000000000000001: Cannot read commit
```

#### 2. Fix Issues

Automatically fix all detected issues:

```bash
# Dry-run (preview changes without applying)
nsha fix --dry-run

# Fix with confirmation prompt
nsha fix

# Fix without confirmation
nsha fix --yes

# Fix with verbose output
nsha fix --verbose

# Fix specific repository
nsha fix --repo /path/to/repo
```

**Example Output:**
```
[STEP 1] Diagnosing repository...
[INFO] Logging to: C:\Users\username\nsha\20260203-105946

[STEP 2] Creating repository backup...
[SUCCESS] Backup created successfully
[SUCCESS] Directory copy backup verified successfully

[STEP 3] Fixing null SHA issues...
[SUCCESS] Fixed 2 issue(s)!
```

#### 3. Verify Repository

Verify that all issues have been resolved:

```bash
# Verify current directory
nsha verify

# Verify specific repository
nsha verify --repo /path/to/repo

# Verbose output
nsha verify --verbose
```

**Example Output:**
```
[STEP 1] Verifying repository integrity...
[SUCCESS] Repository is healthy! No issues found.
```

### Advanced Usage

#### Command Flags

All commands support the following flags:

- `-r, --repo <path>`: Path to Git repository (default: current directory)
- `-v, --verbose`: Enable verbose output with detailed information
- `-h, --help`: Show help for the command

**Fix command additional flags:**
- `--dry-run`: Preview changes without applying them
- `-y, --yes`: Skip confirmation prompt
- `-f, --force`: Force operation even with warnings

#### Complete Workflow Example

```bash
# Step 1: Navigate to your repository
cd /path/to/your/repository

# Step 2: Check for issues
nsha diagnose --verbose

# Step 3: Preview fixes (dry-run)
nsha fix --dry-run --verbose

# Step 4: Apply fixes
nsha fix --yes --verbose

# Step 5: Verify the fix
nsha verify

# Step 6: Push to remote (if needed)
git push origin --force --all
git push origin --force --tags
```

### Logging and Reporting

NSHA automatically creates detailed logs and reports when fixing issues:

**Location:** `~/nsha/YYYYMMDD-HHMMSS/` (user's home directory)

**Files created:**
- `nsha.log` - Detailed operation log
- `report.txt` - Summary of issues found and fixed
- `changes-summary.txt` - List of all changes made
- `backup/repository/` - Complete backup of the repository

**Example:**
```
C:\Users\username\nsha\20260203-105946\
├── nsha.log
├── report.txt
├── changes-summary.txt
└── backup\
    └── repository\
        └── .git\
```

## How It Works

NSHA follows a systematic approach to fix null SHA issues:

### Step 1: Diagnosis
- Runs a comprehensive repository scan (similar to `git fsck`)
- Identifies null SHA references in:
  - Branch references
  - Tag references
  - Packed-refs file
  - Tree objects
  - Commit parents
- Detects missing commits
- Reports all issues found with detailed information

### Step 2: Backup Creation
- Creates a complete backup of the repository
- Attempts git bundle first (preserves full history)
- Falls back to directory copy if bundle fails
- Verifies backup integrity
- Stores backup in user's home directory with timestamp

### Step 3: Cleanup Packed-Refs
- Scans packed-refs file for null SHA entries
- Removes null-like SHA variants (0000000..., 0000000...1, etc.)
- Cleans up duplicate references
- Ensures clean state before fixes

### Step 4: Fix Null SHA Issues
- **References**: Points null SHA references to valid commits
- **Tags**: Updates or removes tags with null SHA
- **Missing Commits**: Fixes references to non-existent commits
- **Tree Objects**: Detects and reports corrupted trees
- Uses git plumbing commands for safe operations

### Step 5: History Rewriting (if needed)
- Implements git-filter-repo functionality in Go
- Walks the entire commit graph
- Rewrites commits with updated parents and trees
- Updates all branches and tags
- Preserves commit metadata (author, date, message)

### Step 6: Garbage Collection
- Runs `git gc --prune=now --aggressive`
- Runs `git prune --expire=now`
- Removes orphaned objects
- Compacts repository

### Step 7: Verification
- Runs final integrity check using `git fsck`
- Confirms all issues are resolved
- Reports any remaining issues

## Project Structure

```
nsha/
├── cmd/                          # CLI commands
│   ├── root.go                  # Root command and shared utilities
│   ├── diagnose.go              # Diagnose command implementation
│   ├── fix.go                   # Fix command implementation
│   └── verify.go                # Verify command implementation
│
├── pkg/                         # Core packages
│   ├── backup/                  # Repository backup functionality
│   │   └── backup.go           # Backup creation and verification
│   ├── git/                     # Git operations
│   │   ├── types.go            # Type definitions and structures
│   │   ├── fsck.go             # Repository scanning and issue detection
│   │   ├── replace.go          # Git replace/graft logic
│   │   ├── filter.go           # History rewriting (filter-repo)
│   │   ├── dryrun.go           # Dry-run analysis and reporting
│   │   └── utils.go            # Utility functions
│   ├── logger/                  # Logging functionality
│   │   └── logger.go           # File and console logging
│   └── report/                  # Report generation
│       └── report.go           # Summary and change reports
│
├── build/                       # Build output (generated)
│   ├── nsha-windows-amd64.exe
│   ├── nsha-linux-amd64
│   ├── nsha-darwin-amd64
│   └── ...
│
├── main.go                      # Application entry point
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
│
├── build.sh                     # Build script (Unix/Linux/macOS)
├── build.ps1                    # Build script (Windows PowerShell)
├── Makefile                     # Make targets for development
│
├── README.md                    # This file
└── LICENSE                      # MIT License
```

### Key Components

#### 1. Command Layer (cmd/)
- **root.go**: Base command, global flags, helper functions for colored output
- **diagnose.go**: Scans repository and reports issues
- **fix.go**: Orchestrates the complete fix process
- **verify.go**: Verifies repository integrity

#### 2. Core Logic (pkg/git/)
- **fsck.go**: Repository scanning using go-git and git fsck
- **replace.go**: Git replace/graft implementation
- **filter.go**: History rewriting (equivalent to git-filter-repo)
- **dryrun.go**: Dry-run analysis with detailed change preview
- **types.go**: Data structures (Issue, BadCommit, DryRunChange, etc.)

#### 3. Support Packages
- **pkg/backup/**: Complete repository backup with verification
- **pkg/logger/**: Structured logging to file and console
- **pkg/report/**: Generate summary and detailed change reports

## Technical Specifications

### Language and Framework
- **Language**: Go 1.21+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Git Library**: go-git v5.4.2 (github.com/go-git/go-git/v5)
- **Terminal Colors**: fatih/color
- **Progress Bars**: schollz/progressbar/v3

### Dependencies

```go
require (
    github.com/go-git/go-git/v5 v5.4.2
    github.com/spf13/cobra v1.8.1
    github.com/fatih/color v1.18.0
    github.com/schollz/progressbar/v3 v3.17.1
)
```

### Architecture

**Design Pattern**: Layered Architecture
- **Presentation Layer**: CLI commands (cmd/)
- **Business Logic Layer**: Git operations (pkg/git/)
- **Data Access Layer**: go-git library + git plumbing commands

**Key Design Decisions**:
1. **Hybrid Approach**: Uses go-git for reading, git plumbing for writing
2. **Safety First**: Always creates backups before modifications
3. **Detailed Logging**: Comprehensive logs for debugging and auditing
4. **Dry-Run Support**: Preview changes before applying
5. **Cross-Platform**: Pure Go with platform-specific build scripts

### Git Operations Used

**Reading Operations** (via go-git):
- Repository opening and validation
- Commit graph traversal
- Reference enumeration
- Tree object inspection

**Writing Operations** (via git plumbing commands):
- `git hash-object -w` - Create new objects
- `git mktree` - Create tree objects
- `git update-ref` - Update references
- `git replace --graft` - Create replacement commits
- `git gc --prune=now --aggressive` - Garbage collection
- `git prune --expire=now` - Remove unreachable objects
- `git bundle create` - Create repository bundle for backup

### Issue Detection

NSHA detects the following types of issues:

1. **Null SHA References** (`IssueTypeNullSHA`)
   - References pointing to `0000000000000000000000000000000000000000`
   - Variants: `0000000000000000000000000000000000000001`, etc.

2. **Missing Commits** (`IssueTypeMissingCommit`)
   - References pointing to non-existent commit objects
   - Dangling references

3. **Broken Parents** (`IssueTypeBrokenParent`)
   - Commits with null SHA parents
   - Commits with missing parent references

4. **Missing Trees** (`IssueTypeMissingTree`)
   - Commits referencing non-existent tree objects
   - Corrupted tree references

5. **Corrupted Trees** (`IssueTypeCorruptedTree`)
   - Tree objects containing null SHA entries
   - Tree objects with invalid file entries

### Performance Characteristics

- **Diagnosis**: O(n) where n = number of commits
- **Fix**: O(n) for history rewriting
- **Memory**: Efficient streaming for large repositories
- **Disk**: Creates backup (size = repository size)

### Platform Support

**Tested Platforms**:
- Windows 10/11 (amd64, arm64)
- Linux (amd64, arm64) - Ubuntu, Debian, CentOS, Fedora
- macOS (amd64, arm64) - Intel and Apple Silicon

**Build Targets**:
- `windows/amd64`, `windows/arm64`
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`

## Building from Source

### Prerequisites
- Go 1.21 or higher
- Git
- Make (optional, for using Makefile)

### Build Commands

```bash
# Simple build
go build -o nsha

# Build with version info
go build -ldflags="-s -w -X main.Version=1.3.0" -o nsha

# Build for all platforms
./build.sh        # Linux/macOS
.\build.ps1       # Windows

# Using Make
make build        # Current platform
make build-all    # All platforms
make clean        # Clean build artifacts
```

### Development Commands

```bash
# Run tests
go test -v ./...
make test

# Format code
go fmt ./...
make fmt

# Run linter
golangci-lint run
make lint

# Install dependencies
go mod download
go mod tidy
make deps
```

## Important Notes

### Before Running

1. **Backup your repository**: NSHA creates automatic backups, but manual backup is recommended
   ```bash
   cp -r /path/to/repo /path/to/repo.backup
   ```

2. **Understand the impact**: This tool may rewrite Git history, which means:
   - Commit hashes may change
   - You may need to force-push to remote repositories
   - Collaborators may need to re-clone or reset their local copies

3. **Coordinate with team**: If working in a team, coordinate before running this tool

4. **Test with dry-run**: Always use `--dry-run` first to preview changes
   ```bash
   nsha fix --dry-run --verbose
   ```

### After Running

1. **Review changes**: Check the repository with `git log` and `git fsck`
   ```bash
   git log --oneline --graph --all
   git fsck --full
   ```

2. **Check logs**: Review the detailed logs in `~/nsha/YYYYMMDD-HHMMSS/`
   - `nsha.log` - Complete operation log
   - `report.txt` - Summary of fixes
   - `changes-summary.txt` - Detailed changes

3. **Push to remote** (if history was rewritten): Force-push the fixed history
   ```bash
   git push origin --force --all
   git push origin --force --tags
   ```

4. **Notify collaborators**: Team members should re-clone or reset their repositories
   ```bash
   # Option 1: Re-clone
   git clone <repository-url>

   # Option 2: Reset (CAUTION: loses local changes)
   git fetch origin
   git reset --hard origin/main
   ```

### Safety Features

- **Automatic Backups**: Complete repository backup before any modifications
- **Backup Verification**: Validates backup integrity before proceeding
- **Dry-Run Mode**: Preview all changes without applying them
- **Detailed Logging**: Complete audit trail of all operations
- **Confirmation Prompts**: Asks for confirmation before destructive operations
- **Rollback Support**: Backup can be restored if needed

## Troubleshooting

### Common Issues

#### 1. "No issues found" but git fsck shows errors

**Solution**: Try verbose mode to see more details
```bash
nsha diagnose --verbose
```

Some issues may not be detected by NSHA's current implementation. Check the logs for details.

#### 2. Build fails with "go: command not found"

**Solution**: Install Go from https://go.dev/dl/

Verify installation:
```bash
go version
```

#### 3. After fix, can't push to remote

**Solution**: You need to force push (history was rewritten)
```bash
git push origin --force --all
git push origin --force --tags
```

**Warning**: Force push overwrites remote history. Coordinate with your team first.

#### 4. Team members can't pull after fix

**Solution**: They need to re-clone or reset their local repository

**Option 1: Re-clone (Recommended)**
```bash
cd ..
rm -rf old-repo
git clone <repository-url>
```

**Option 2: Reset (CAUTION: loses local changes)**
```bash
git fetch origin
git reset --hard origin/main
```

#### 5. "Backup verification failed" message

**Cause**: Git bundle creation failed, tool fell back to directory copy

**Solution**: This is normal behavior. The directory copy backup is complete and safe. The message is informational only.

#### 6. Dry-run shows issues remain

**Cause**: Dry-run doesn't actually fix anything, it only previews changes

**Solution**: This is expected behavior. Run without `--dry-run` to apply fixes:
```bash
nsha fix --yes
```

#### 7. Permission denied when building

**Linux/macOS Solution**:
```bash
chmod +x build.sh
./build.sh
```

**Windows Solution**: Run PowerShell as Administrator
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
.\build.ps1
```

#### 8. Repository still broken after fix

**Diagnosis**:
```bash
# Check what issues remain
git fsck --full

# Check NSHA logs
cat ~/nsha/YYYYMMDD-HHMMSS/nsha.log
```

**Solution**: Some complex issues may require manual intervention. Check the logs for details and consider:
- Running `nsha fix` again
- Manually fixing remaining issues
- Restoring from backup and trying alternative approaches

#### 9. Out of memory errors

**Cause**: Very large repositories may consume significant memory

**Solution**:
- Close other applications
- Increase system swap/page file
- Run on a machine with more RAM
- Process repository in smaller chunks (advanced)

#### 10. Logs not created

**Cause**: Logs are only created when issues are found and fixed

**Solution**: This is normal behavior. If no issues are found, no logs are created.

### Getting Help

If you encounter issues not covered here:

1. **Check the logs**: `~/nsha/YYYYMMDD-HHMMSS/nsha.log`
2. **Run with verbose**: `nsha fix --verbose`
3. **Check Git status**: `git fsck --full`
4. **Review backup**: Check `~/nsha/YYYYMMDD-HHMMSS/backup/`

## Development

### Setting Up Development Environment

```bash
# Clone the repository
git clone https://github.com/RahulGS02/nsha-tool.git
cd nsha-tool

# Install dependencies
go mod download

# Build
go build -o nsha

# Run tests
go test -v ./...
```

### Project Development Commands

```bash
# Format code
make fmt
go fmt ./...

# Run linter
make lint
golangci-lint run

# Run tests
make test
go test -v ./...

# Build for current platform
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

### Code Structure

**Adding a new command**:
1. Create new file in `cmd/` (e.g., `cmd/newcommand.go`)
2. Define command using Cobra
3. Add command to root in `cmd/root.go`

**Adding new Git operations**:
1. Add functions to `pkg/git/`
2. Use go-git for reading operations
3. Use git plumbing commands for writing operations

**Adding new issue types**:
1. Define new issue type in `pkg/git/types.go`
2. Add detection logic in `pkg/git/fsck.go`
3. Add fix logic in `pkg/git/replace.go` or `pkg/git/filter.go`

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v ./pkg/git -run TestFsck
```

### Contributing

Contributions are welcome! Please follow these guidelines:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/my-feature`
3. **Make your changes**
4. **Add tests** for new functionality
5. **Run tests**: `go test ./...`
6. **Format code**: `go fmt ./...`
7. **Commit changes**: `git commit -am 'Add new feature'`
8. **Push to branch**: `git push origin feature/my-feature`
9. **Create Pull Request**

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions small and focused
- Write tests for new functionality

## Version History

### v1.3.0 (Current)
- Renamed `.nsha` to `nsha` directory
- Logs saved in user's home directory with timestamp
- Backup entire repository folder (not just .git)
- Enhanced dry-run output with detailed changes
- Fixed confusing verification messages
- Improved backup verification for directory copies

### v1.2.0
- Added comprehensive logging system
- Added repository backup system
- Added detailed reporting system
- Conditional initialization (only when issues exist)
- Merged report files (reduced from 3 to 2)

### v1.1.0
- Complete tag fixing functionality
- Missing commit handling
- Tree corruption detection
- Automatic garbage collection
- Enhanced packed-refs cleanup

### v1.0.0
- Initial release
- Basic null SHA detection and fixing
- Diagnose, fix, and verify commands
- Cross-platform support

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Resources

- [Git Internals](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain)
- [Git Replace](https://git-scm.com/docs/git-replace)
- [Git Filter-Repo](https://github.com/newren/git-filter-repo)
- [go-git Documentation](https://pkg.go.dev/github.com/go-git/go-git/v5)
- [Cobra CLI Framework](https://github.com/spf13/cobra)

## Website Deployment (Netlify)

The NSHA website is automatically deployed to Netlify via GitHub Actions.

### Setup Netlify Deployment

1. **Create Netlify Account**
   - Sign up at https://netlify.com (free)
   - Connect with GitHub

2. **Create New Site**
   - Import your NSHA repository
   - Build settings:
     - Branch: `main`
     - Publish directory: `docs`
   - Deploy site

3. **Get Credentials**
   - Site ID: Found in Site settings
   - Auth Token: User settings → Applications → New access token

4. **Add GitHub Secrets**
   - Go to repository Settings → Secrets → Actions
   - Add `NETLIFY_AUTH_TOKEN` (your personal access token)
   - Add `NETLIFY_SITE_ID` (your site ID)

5. **Deploy**
   ```bash
   git push origin main
   # Website auto-deploys to Netlify!
   ```

### Website Features

- **Live Demo** - Interactive terminal playground
- **Download Statistics** - Real-time GitHub stats
- **Comments** - Utterances (GitHub Issues)
- **OS Detection** - Auto-detects user's platform
- **Responsive Design** - Mobile, tablet, desktop

### Website URL

- Default: `https://your-site-name.netlify.app`
- Custom domain: Configure in Netlify dashboard

## Author

Created by Rahul

---

**If you find this tool helpful, please consider giving it a star!**

