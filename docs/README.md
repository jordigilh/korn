# Korn Documentation

This directory contains comprehensive documentation for Korn. Each file focuses on a specific aspect of using and contributing to Korn.

## Documentation Structure

### For Users

| File | Purpose | Best For |
|------|---------|----------|
| **[onboarding.md](onboarding.md)** | Complete setup guide with label explanations | First-time users setting up Korn |
| **[commands.md](commands.md)** | Full command reference with examples | Daily usage and scripting |
| **[validation-rules.md](validation-rules.md)** | Understanding validations and debugging | Troubleshooting validation failures |
| **[examples.md](examples.md)** | Practical workflows and advanced usage | Learning best practices |

### For Contributors

| File | Purpose | Best For |
|------|---------|----------|
| **[contributing.md](contributing.md)** | Development setup and guidelines | Contributing code or documentation |

## Quick Navigation

### Getting Started
1. **New to Korn?** → Start with [onboarding.md](onboarding.md)
2. **Need to label resources?** → See [onboarding.md#why-labels-are-required](onboarding.md#why-labels-are-required)
3. **Ready to create releases?** → Check [examples.md#release-workflows](examples.md#release-workflows)

### Daily Usage
- **Command syntax help** → [commands.md](commands.md)
- **Common workflows** → [examples.md](examples.md)
- **Troubleshooting** → [validation-rules.md#validation-failures](validation-rules.md#validation-failures)

### Advanced Topics
- **Validation logic** → [validation-rules.md](validation-rules.md)
- **Automation scripts** → [examples.md#advanced-workflows](examples.md#advanced-workflows)
- **Contributing** → [contributing.md](contributing.md)

## Documentation Overview

### [Onboarding Guide](onboarding.md) (11KB)
Complete setup process including:
- Why labels are required and how they work
- Step-by-step labeling instructions
- Bundle Dockerfile configuration
- Validation workflow explanation

### [Commands Reference](commands.md) (7.4KB)
Comprehensive command documentation:
- All available commands with flags
- Usage examples and patterns
- Common workflows and debugging

### [Validation Rules](validation-rules.md) (6.1KB)
Understanding Korn's validation logic:
- Operator vs FBC validation differences
- Common failure scenarios and fixes
- Best practices for validation success

### [Examples & Workflows](examples.md) (11KB)
Practical usage examples:
- Complete setup workflows
- Release automation scripts
- Troubleshooting procedures
- Advanced usage patterns

### [Contributing Guide](contributing.md) (8.3KB)
Development information:
- Environment setup and build process
- Code structure and guidelines
- Testing and documentation standards
- Pull request process

## Quick Reference

### Essential Commands
```bash
# Setup verification
korn get application
korn get component --app operator-1-0

# Release workflow
SNAPSHOT=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')
korn create release --app operator-1-0 --environment staging --snapshot $SNAPSHOT

# Debugging
korn get snapshot --app operator-1-0
korn get releaseplan --app operator-1-0
```

### Common Issues
- **No valid candidate found** → [validation-rules.md#validation-failures](validation-rules.md#validation-failures)
- **Bundle validation failures** → [examples.md#troubleshooting](examples.md#troubleshooting-common-issues)
- **Missing labels** → [onboarding.md#what-breaks-without-labels](onboarding.md#what-breaks-without-labels)

## Contributing to Documentation

When updating documentation:

1. **Keep user focus**: Write for the intended audience
2. **Include examples**: Show practical usage
3. **Link between docs**: Cross-reference related information
4. **Test examples**: Ensure code examples work
5. **Update this index**: Keep navigation current

See [contributing.md](contributing.md) for detailed guidelines.