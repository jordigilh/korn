# Commands Reference

Korn provides commands for getting information about Konflux resources, creating releases, and waiting for completion.

## Command Structure

All Korn commands follow this structure:
```bash
korn [GLOBAL FLAGS] <command> <subcommand> [FLAGS] [ARGUMENTS]
```

## Global Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--namespace` | Override current namespace | `--namespace my-operator-namespace` |
| `--help` | Show help for any command | `korn --help` |

## Namespace Handling

All Korn commands operate within a Kubernetes namespace context. By default, Korn uses the current namespace from your Kubernetes configuration (the namespace set in your current context). You can override this behavior using the global `--namespace` flag:

```bash
# Use current namespace from kubectl context (default)
korn get application

# Override to specific namespace
korn get application --namespace my-operator-namespace

# Set namespace for all commands in session
kubectl config set-context --current --namespace=my-operator-namespace
```

This ensures that Korn operations are scoped to the appropriate namespace where your Konflux resources are deployed.

## Get Commands

### get application

List all applications with their types.

```bash
korn get application [FLAGS]
```

**Examples:**
```bash
# List all applications
korn get application

# List applications in specific namespace
korn get application --namespace my-namespace
```

**Output:**
```
NAME           TYPE       AGE
fbc-v4-15      fbc        59d
operator-1-0   operator   66d
```

### get component

List components for an application.

```bash
korn get component [COMPONENT_NAME] [FLAGS]
```

**Flags:**
| Flag | Alias | Description | Example |
|------|-------|-------------|---------|
| `--application` | `--app` | Filter by application name | `--app operator-1-0` |

**Examples:**
```bash
# List all components
korn get component

# List components for specific application
korn get component --app operator-1-0

# Get specific component
korn get component operator-bundle-1-0
```

### get snapshot

Get snapshots for validation and inspection.

```bash
korn get snapshot [SNAPSHOT_NAME] [FLAGS]
```

**Flags:**
| Flag | Alias | Description | Example |
|------|-------|-------------|---------|
| `--application` | `--app` | Filter by application name | `--app operator-1-0` |
| `--sha` | - | Get snapshot by commit SHA | `--sha abc123...` |
| `--version` | - | Get snapshot by version | `--version v1.0.15` |
| `--candidate` | `-c` | Get latest valid candidate | `--candidate` |

**Examples:**
```bash
# List all snapshots
korn get snapshot

# Get snapshots for application
korn get snapshot --app operator-1-0

# Get latest release candidate
korn get snapshot --app operator-1-0 --candidate

# Get snapshot by version
korn get snapshot --app operator-1-0 --version v1.0.15
```

> **Note:** The `--version` and `--candidate` flags cannot be used together. If both are specified, `--version` takes precedence.

### get release

List releases for an application.

```bash
korn get release [RELEASE_NAME] [FLAGS]
```

**Flags:**
| Flag | Alias | Description | Example |
|------|-------|-------------|---------|
| `--application` | `--app` | Filter by application name | `--app operator-1-0` |

**Examples:**
```bash
# List all releases
korn get release

# List releases for application
korn get release --app operator-1-0

# Get specific release
korn get release my-release-name
```

### get releaseplan

List release plans for an application.

```bash
korn get releaseplan [RELEASEPLAN_NAME] [FLAGS]
```

**Flags:**
| Flag | Alias | Description | Example |
|------|-------|-------------|---------|
| `--application` | `--app` | Filter by application name | `--app operator-1-0` |

**Examples:**
```bash
# List all release plans
korn get releaseplan

# List release plans for application
korn get releaseplan --app operator-1-0

# Get specific release plan
korn get releaseplan operator-staging-1-0
```

## Create Commands

### create release

Create a new release from a validated snapshot.

```bash
korn create release [FLAGS]
```

**Flags:**
| Flag | Alias | Description | Default | Example |
|------|-------|-------------|---------|---------|
| `--application` | `--app` | Application name for the release | - | `--app operator-1-0` |
| `--environment` | `--env` | Target environment (`staging` or `production`) | `staging` | `--environment production` |
| `--snapshot` | - | Use specific snapshot instead of latest candidate | - | `--snapshot snapshot-xyz123` |
| `--sha` | - | Use snapshot associated with specific commit SHA | - | `--sha abc1234def5678` |
| `--releaseNotes` | `--rn` | Path to YAML file containing release notes | - | `--releaseNotes release-notes.yaml` |
| `--dryrun` | - | Output manifest without creating release | `false` | `--dryrun` |
| `--wait` | `-w` | Wait for release completion | `true` | `--wait=false` |
| `--force` | `-f` | Force creation even if snapshot was used before | `false` | `--force` |
| `--output` | `-o` | Output format (`json` or `yaml`) | - | `--output yaml` |
| `--timeout` | `-t` | Timeout in minutes for wait operation | `60` | `--timeout 120` |

> **Note:** `--dryrun` and `--wait` flags are mutually exclusive.

**Examples:**
```bash
# Simple staging release
korn create release --app operator-1-0 --environment staging

# Production release with release notes
korn create release --app operator-1-0 --environment production --releaseNotes release-notes.yaml

# Force release (retry failed release)
korn create release --app operator-1-0 --environment staging --force

# Dry run to see manifest
korn create release --app operator-1-0 --environment staging --dryrun --output yaml
```

## Wait Commands

### waitfor release

Wait for a release to complete.

```bash
korn waitfor release <RELEASE_NAME> [FLAGS]
```

**Flags:**
| Flag | Alias | Description | Default | Example |
|------|-------|-------------|---------|---------|
| `--timeout` | `-t` | Timeout in minutes | `60` | `--timeout 120` |

**Examples:**
```bash
# Wait for release with default timeout
korn waitfor release my-release-abc123

# Wait with custom timeout
korn waitfor release my-release-abc123 --timeout 180
```

## Common Patterns

### Validation Workflow
```bash
# 1. Check application setup
korn get application

# 2. Verify components are labeled
korn get component --app operator-1-0

# 3. Check release plans
korn get releaseplan --app operator-1-0

# 4. Get latest candidate snapshot
korn get snapshot --app operator-1-0 --candidate

# 5. Create release
korn create release --app operator-1-0 --environment staging
```

### Release Workflow
```bash
# 1. Get latest candidate
SNAPSHOT=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')

# 2. Create staging release
korn create release --app operator-1-0 --environment staging --snapshot $SNAPSHOT

# 3. After validation, promote to production
korn create release --app operator-1-0 --environment production --snapshot $SNAPSHOT
```

### Debugging Workflow
```bash
# Check all snapshots for application
korn get snapshot --app operator-1-0

# Find snapshot by commit
korn get snapshot --app operator-1-0 --sha abc1234

# Check specific snapshot status
korn get snapshot snapshot-name-xyz123

# List recent releases
korn get release --app operator-1-0
```

## Getting Help

For detailed help on any command:
```bash
korn --help                    # Global help
korn get --help               # Help for get commands
korn create release --help    # Help for specific command
```