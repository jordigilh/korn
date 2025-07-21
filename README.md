# Korn

[![Go Report Card](https://goreportcard.com/badge/github.com/jordigilh/korn)](https://goreportcard.com/report/github.com/jordigilh/korn)

**Korn** is an opinionated CLI tool for releasing Operators with Konflux, providing automated validations and streamlined release workflows.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Operator Onboarding](#operator-onboarding)
- [Get Snapshot](#get-snapshot)
- [Release Process](#release-process)
- [Validation Rules](#validation-rules)
- [Examples](#examples)
- [Contributing](#contributing)

## Overview

Releasing an operator with Konflux involves complex validations and checks across multiple domain constructs (snapshots, releases, release plans, and release plan admissions). While Konflux enforces its own validations through Enterprise Contract Plans and RPA rules, it's up to the operator's release team to ensure artifact consistency.

**Korn simplifies this process by automating critical validations:**

- ✅ Snapshot creation from push events
- ✅ Snapshot success validation
- ✅ Component image existence verification
- ✅ CSV/bundle image reference consistency
- ✅ Version label matching across components
- ✅ Release notes population

Any snapshot failing these validations cannot be used for release.

## Installation

### From Source

```bash
git clone https://github.com/jordigilh/korn.git
cd korn
make build
cp output/korn /usr/local/bin/  # or any directory in your PATH
```

### Prerequisites

- Go 1.21+ (for building from source)
- Kubernetes cluster access with Konflux installed
- Appropriate RBAC permissions for Konflux resources

## Quick Start

1. **Label your applications:**
   ```bash
   oc label application operator-1-0 korn.redhat.io/application=operator
   oc label application fbc-v4-15 korn.redhat.io/application=fbc
   ```

2. **Label your components:**
   ```bash
   oc label component operator-bundle-1-0 korn.redhat.io/component=bundle
   oc label component controller-rhel9-operator-1-0 korn.redhat.io/bundle-label=controller-rhel9-operator
   ```

3. **Label your release plans:**
   ```bash
   oc label releaseplan operator-staging-1-0 korn.redhat.io/environment=staging
   oc label releaseplan operator-production-1-0 korn.redhat.io/environment=production
   ```

4. **Create a release:**
   ```bash
   korn create release -app operator-1-0 -environment staging -releaseNotes releaseNotes-1.0.1.yaml
   ```

## Commands

### Get Commands

| Command | Description | Example |
|---------|-------------|---------|
| `get application` | List all applications with their types | `korn get application` |
| `get component` | List components for an application | `korn get component -app operator-1-0` |
| `get snapshot` | Get latest valid snapshot for an application | `korn get snapshot -app operator-1-0` |
| `get release` | List releases for an application | `korn get release -app operator-1-0` |
| `get releaseplan` | List release plans for an application | `korn get releaseplan -app operator-1-0` |

### Create Commands

| Command | Description | Example |
|---------|-------------|---------|
| `create release` | Create a new release | `korn create release -app operator-1-0 -environment staging` |

### Wait Commands

| Command | Description | Example |
|---------|-------------|---------|
| `waitfor release` | Wait for release completion | `korn waitfor release -name my-release` |

For detailed command options:
```bash
korn <command> -h
```

## Operator Onboarding

To use Korn with your operator, you need to label existing Konflux resources appropriately.

### 1. Application Labels

Label applications to distinguish between operator and FBC (File Based Catalog) types:

```bash
# For operator applications
oc label application operator-1-0 korn.redhat.io/application=operator
oc label application operator-1-1 korn.redhat.io/application=operator

# For FBC applications
oc label application fbc-v4-15 korn.redhat.io/application=fbc
oc label application fbc-v4-16 korn.redhat.io/application=fbc
```

**Verify labeling:**
```bash
korn get application
```

Expected output:
```
NAME           TYPE       AGE
fbc-v4-15      fbc        59d
fbc-v4-16      fbc        59d
operator-1-0   operator   66d
operator-1-1   operator   66d
```

### 2. Component Labels

#### Bundle Component
Identify the bundle component in operator applications:

```bash
oc label component operator-bundle-1-0 korn.redhat.io/component=bundle
```

#### Bundle Label Mapping
For each component referenced in the bundle, specify the label name used in the bundle's Dockerfile:

```bash
oc label component controller-rhel9-operator-1-0 korn.redhat.io/bundle-label=controller-rhel9-operator
oc label component console-plugin-1-0 korn.redhat.io/bundle-label=console-plugin
```

**Verify component labeling:**
```bash
korn get component -app operator-1-0
```

Expected output:
```
NAME                            TYPE     BUNDLE LABEL                AGE
console-plugin-1-0                       console-plugin              67d
controller-rhel9-operator-1-0            controller-rhel9-operator   67d
operator-bundle-1-0             bundle                               67d
```

### 3. Release Plan Labels

Label release plans to identify target environments:

```bash
# Staging plans
oc label releaseplan operator-staging-1-0 korn.redhat.io/environment=staging
oc label releaseplan fbc-v4-15-release-as-staging-fbc korn.redhat.io/environment=staging

# Production plans
oc label releaseplan operator-production-1-0 korn.redhat.io/environment=production
oc label releaseplan fbc-v4-15-release-as-production-fbc korn.redhat.io/environment=production
```

**Verify release plan labeling:**
```bash
korn get releaseplan -app operator-1-0
```

Expected output:
```
NAME                      APPLICATION    ENVIRONMENT   RELEASE PLAN ADMISSION              ACTIVE   AGE
operator-staging-1-0      operator-1-0   staging       rhtap-releng-tenant/my-operator-staging-1-0   true     66d
operator-production-1-0   operator-1-0   production    rhtap-releng-tenant/my-operator-prod-1-0      true     66d
```

### 4. Update Bundle Dockerfile

Your bundle's Dockerfile must include labels that map component names to their image digests:

```dockerfile
FROM scratch

ARG VERSION=1.0

# Component image labels - these must match your component bundle-label values
LABEL controller-rhel9-operator="registry.stage.redhat.io/my-operator-tech-preview/my-rhel9-operator@sha256:6b33780302d877c80f3775673aed629975e6bebb8a8bd3499a9789bd44d04861"
LABEL console-plugin="registry.stage.redhat.io/my-operator-tech-preview/my-console-plugin-rhel9@sha256:723276c6a1441d6b0d13b674b905385deec0291ac458260a569222b5612f73c4"

COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
COPY bundle/tests/scorecard /tests/scorecard/
COPY LICENSE /licenses/licenses
```

> **Note:** Consider using automated tools like nudges to keep these labels synchronized with actual image digests.

## Get Snapshot

The `get snapshot` command allows you to retrieve snapshots for validation and inspection before creating releases.

### Command Syntax

```bash
korn get snapshot [SNAPSHOT_NAME] [FLAGS]
```

#### Available Flags

| Flag | Alias | Description | Example |
|------|-------|-------------|---------|
| `--application` | `--app` | Application name to filter snapshots | `--app operator-1-0` |
| `--sha` | - | Get snapshot associated with specific commit SHA | `--sha 245fca6109a1f32e5ded0f7e330a85401aa2704a` |
| `--version` | - | Get latest snapshot matching version in bundle's label | `--version v0.0.11` |
| `--candidate` | `-c` | Filter snapshots suitable for next release | `--candidate` |

#### Basic Examples

**List all snapshots in namespace:**
```bash
korn get snapshot
```

**Get snapshots for specific application:**
```bash
korn get snapshot --app operator-1-0
```

**Get latest release candidate snapshot:**
```bash
korn get snapshot --app operator-1-0 --candidate
```

**Get specific snapshot by name:**
```bash
korn get snapshot snapshot-sample-xyz123
```

#### Advanced Examples

**Get snapshot by commit SHA:**
```bash
korn get snapshot \
  --app operator-1-0 \
  --sha 245fca6109a1f32e5ded0f7e330a85401aa2704a
```

**Get latest snapshot for specific version:**
```bash
korn get snapshot \
  --app operator-1-0 \
  --version v1.0.15
```

**Get release candidate with application filter:**
```bash
korn get snapshot \
  --app operator-1-0 \
  --candidate
```

#### Output Format

The command outputs a table with the following columns:

| Column | Description |
|--------|-------------|
| **Name** | Snapshot name |
| **Application** | Associated application name |
| **SHA** | Git commit SHA |
| **Commit** | Commit message title |
| **Status** | Test status (Succeeded/Failed/Pending) |
| **Age** | Time since snapshot creation |

#### Example Output

```
NAME                           APPLICATION    SHA      COMMIT                    STATUS     AGE
snapshot-sample-xyz123         operator-1-0   abc123   Fix security vulnerability Succeeded  2d
snapshot-sample-def456         operator-1-0   def456   Update dependencies       Failed     1d
snapshot-sample-ghi789         operator-1-0   ghi789   Add new feature          Succeeded  12h
```

#### Use Cases

**Pre-release validation:**
```bash
# Check latest candidate before creating release
korn get snapshot --app operator-1-0 --candidate

# Verify specific snapshot status
korn get snapshot snapshot-sample-xyz123
```

**Debugging and troubleshooting:**
```bash
# Find snapshot for specific commit
korn get snapshot --app operator-1-0 --sha abc1234def5678

# Check all snapshots for application
korn get snapshot --app operator-1-0
```

**Version-specific releases:**
```bash
# Find snapshot for specific version
korn get snapshot --app operator-1-0 --version v1.0.15

# Use in release creation
korn create release --app operator-1-0 --environment staging --snapshot $(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')
```

## Release Process

### Creating a Release

The primary command for releasing is:

```bash
korn create release [FLAGS]
```

#### Available Flags

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

#### Basic Examples

**Simple staging release:**
```bash
korn create release --app operator-1-0 --environment staging
```

**Production release with release notes:**
```bash
korn create release --app operator-1-0 --environment production --releaseNotes releaseNotes-1.0.1.yaml
```

**Quick release without waiting:**
```bash
korn create release --app operator-1-0 --environment staging --wait=false
```

#### Advanced Examples

**Release with specific snapshot:**
```bash
korn create release \
  --app operator-1-0 \
  --environment staging \
  --snapshot snapshot-sample-xyz123 \
  --releaseNotes release-notes.yaml
```

**Release using specific commit SHA:**
```bash
korn create release \
  --app operator-1-0 \
  --environment production \
  --sha abc1234def5678901234567890abcdef12345678 \
  --releaseNotes security-release-notes.yaml \
  --timeout 180
```

**Force release (retry failed release):**
```bash
korn create release \
  --app operator-1-0 \
  --environment staging \
  --force \
  --releaseNotes updated-release-notes.yaml
```

**Dry run - generate manifest without creating:**
```bash
korn create release \
  --app operator-1-0 \
  --environment staging \
  --releaseNotes release-notes.yaml \
  --dryrun \
  --output yaml
```

**Generate JSON manifest:**
```bash
korn create release \
  --app operator-1-0 \
  --environment production \
  --output json \
  --dryrun > release-manifest.json
```

**Extended timeout for long-running releases:**
```bash
korn create release \
  --app operator-1-0 \
  --environment production \
  --releaseNotes release-notes.yaml \
  --timeout 300 \
  --wait
```

### Release Notes Format

Korn supports embedding release notes from YAML files. See example formats:
- [Bug release notes](test-data/releaseNotes.rhba)
- [Security release notes](test-data/releaseNotes.rhsa)

For detailed release notes structure, see [Konflux documentation](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html#release).

### Validation Process

Before creating a release, Korn:

1. **Finds the latest snapshot** for the specified application
2. **Validates the snapshot** meets all criteria
3. **Checks image consistency** between snapshot and bundle
4. **Verifies version labels** across components
5. **Creates the release object** with the validated snapshot

## Validation Rules

### Operator Applications

- ✅ Snapshot created from push event
- ✅ Snapshot marked as successful
- ✅ All component images exist and are accessible
- ✅ Bundle CSV references match snapshot image specs
- ✅ Version labels consistent across all components

### FBC Applications

- ✅ Snapshot marked as successful
- ✅ Container image exists and is accessible

> **Note:** FBC releases require manual catalog updates via PR - Korn assists with snapshot validation only.

## Examples

### Complete Onboarding Example

```bash
# 1. Label applications
oc label application operator-1-0 korn.redhat.io/application=operator
oc label application fbc-v4-15 korn.redhat.io/application=fbc

# 2. Label components
oc label component operator-bundle-1-0 korn.redhat.io/component=bundle
oc label component controller-rhel9-operator-1-0 korn.redhat.io/bundle-label=controller-rhel9-operator
oc label component console-plugin-1-0 korn.redhat.io/bundle-label=console-plugin

# 3. Label release plans
oc label releaseplan operator-staging-1-0 korn.redhat.io/environment=staging
oc label releaseplan operator-production-1-0 korn.redhat.io/environment=production

# 4. Verify setup
korn get application
korn get component -app operator-1-0
korn get releaseplan -app operator-1-0

# 5. Check latest snapshot
korn get snapshot -app operator-1-0

# 6. Create release
korn create release -app operator-1-0 -environment staging -releaseNotes release-notes.yaml
```

### Release Workflow Example

```bash
# Check available applications
korn get application

# Examine specific application components
korn get component -app operator-1-0

# Get the latest valid snapshot
korn get snapshot -app operator-1-0

# Review release plans
korn get releaseplan -app operator-1-0

# Create staging release
korn create release -app operator-1-0 -environment staging

# Wait for release completion (optional)
korn waitfor release -name <release-name>

# After staging validation, promote to production
korn create release -app operator-1-0 -environment production
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Commit your changes: `git commit -m 'Add amazing feature'`
5. Push to the branch: `git push origin feature/amazing-feature`
6. Open a Pull Request

### Development Setup

```bash
git clone https://github.com/jordigilh/korn.git
cd korn
go mod download
make build
make test
```

## License

This project is licensed under the [LICENSE](LICENSE) file in the repository.

---

**Need help?** Check the command help: `korn <command> -h` or review the [examples](#examples) section.
