# Korn

[![Go Report Card](https://goreportcard.com/badge/github.com/jordigilh/korn)](https://goreportcard.com/report/github.com/jordigilh/korn)

**Korn** is an opinionated CLI tool for releasing Operators with Konflux, providing automated validations and streamlined release workflows.

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

## Quick Start

### Installation

```bash
git clone https://github.com/jordigilh/korn.git
cd korn
make build
cp output/korn /usr/local/bin/
```

### Prerequisites

- Go 1.21+ (for building from source)
- Kubernetes cluster access with Konflux installed
- Appropriate RBAC permissions for Konflux resources

### Basic Usage

1. **Set up your operator** (see [Onboarding Guide](docs/onboarding.md)):
   ```bash
   # Label applications, components, and release plans
   oc label application operator-1-0 korn.redhat.io/application=operator
   oc label component operator-bundle-1-0 korn.redhat.io/component=bundle
   oc label releaseplan operator-staging-1-0 korn.redhat.io/environment=staging
   ```

2. **Validate setup**:
   ```bash
   korn get application
   korn get component --app operator-1-0
   ```

3. **Create a release**:
   ```bash
   # Get latest candidate snapshot
   SNAPSHOT=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')

   # Create release with snapshot
   korn create release --app operator-1-0 --environment staging --snapshot $SNAPSHOT
   ```

## Essential Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `get application` | List applications with types | `korn get application` |
| `get snapshot --candidate` | Get latest valid snapshot | `korn get snapshot --app operator-1-0 --candidate` |
| `create release` | Create new release | `korn create release --app operator-1-0 --environment staging --snapshot <snapshot-name>` |
| `waitfor release` | Wait for completion | `korn waitfor release <release-name>` |

For complete command reference, see [Commands Documentation](docs/commands.md).

## Namespace Handling

Korn uses your current Kubernetes namespace by default. Override with `--namespace`:

```bash
# Use current namespace
korn get application

# Override namespace
korn get application --namespace my-operator-namespace
```

## Documentation

| Topic | Description | Link |
|-------|-------------|------|
| **Getting Started** | Complete onboarding process | [Onboarding Guide](docs/onboarding.md) |
| **Commands** | Full command reference with examples | [Commands Reference](docs/commands.md) |
| **Validation** | Understanding validation rules and debugging | [Validation Rules](docs/validation-rules.md) |
| **Workflows** | Practical examples and advanced workflows | [Examples & Workflows](docs/examples.md) |
| **Contributing** | Development setup and contribution guidelines | [Contributing Guide](docs/contributing.md) |

## Example Workflow

```bash
# 1. Verify setup
korn get application
korn get component --app operator-1-0

# 2. Check latest candidate and capture snapshot name
SNAPSHOT=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')
echo "Using snapshot: $SNAPSHOT"

# 3. Create staging release with captured snapshot
korn create release --app operator-1-0 --environment staging --snapshot $SNAPSHOT

# 4. Promote to production using same snapshot
korn create release --app operator-1-0 --environment production --snapshot $SNAPSHOT
```

## Getting Help

```bash
# Command help
korn --help
korn create release --help

# Check command reference
open docs/commands.md

# Review examples
open docs/examples.md
```

## Key Features

- **Automated Validation**: Comprehensive checks for operator snapshots
- **Environment Targeting**: Simple staging/production workflows
- **Bundle Verification**: Ensures CSV references match snapshot images
- **Release Notes Integration**: YAML-based release documentation
- **Konflux Integration**: Native support for Konflux resources and labels

## Prerequisites for Usage

Before using Korn, ensure you have:

1. **Labeled Konflux Resources**: Applications, components, and release plans must be properly labeled
2. **Bundle Configuration**: Bundle Dockerfile must include component image labels
3. **Kubernetes Access**: Valid kubeconfig with appropriate permissions
4. **Konflux Setup**: Working Konflux installation with your operator onboarded

See the [Onboarding Guide](docs/onboarding.md) for detailed setup instructions.

## Support

- **Documentation**: Check the [docs/](docs/) directory for comprehensive guides
- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/jordigilh/korn/issues)
- **Examples**: See [examples and workflows](docs/examples.md) for common use cases

## License

This project is licensed under the [LICENSE](LICENSE) file in the repository.
