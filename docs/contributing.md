# Contributing to Korn

We welcome contributions to Korn! This guide will help you get started with contributing to the project.

## Development Setup

### Prerequisites

- Go 1.21+
- Kubernetes cluster access with Konflux installed
- `kubectl` and `oc` CLI tools
- Git
- Make

### Environment Setup

1. **Fork and clone the repository:**
   ```bash
   git clone https://github.com/your-username/korn.git
   cd korn
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Build the project:**
   ```bash
   make build
   ```

4. **Run tests:**
   ```bash
   make test
   ```

5. **Verify installation:**
   ```bash
   ./output/korn --help
   ```

### Development Workflow

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/amazing-feature
   ```

2. **Make your changes**
   - Follow Go best practices
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes:**
   ```bash
   # Run unit tests
   make test

   # Build and test manually
   make build
   ./output/korn get application
   ```

4. **Commit your changes:**
   ```bash
   git add .
   git commit -m 'Add amazing feature'
   ```

5. **Push to your fork:**
   ```bash
   git push origin feature/amazing-feature
   ```

6. **Open a Pull Request**

## Code Structure

```
korn/
├── cmd/                    # CLI command implementations
│   ├── create/
│   │   └── release/
│   ├── get/
│   │   ├── application/
│   │   ├── component/
│   │   ├── release/
│   │   ├── releaseplan/
│   │   └── snapshot/
│   └── waitfor/
│       └── release/
├── internal/               # Internal packages
│   └── konflux/           # Konflux API interactions
├── docs/                  # Documentation
├── test-data/             # Test data files
└── vendor/                # Vendored dependencies
```

### Key Components

- **CLI Commands** (`cmd/`): Individual command implementations using urfave/cli
- **Konflux Package** (`internal/konflux/`): Core business logic for Konflux operations
- **Types** (`internal/konflux/types.go`): Data structures and interfaces
- **Validation Logic** (`internal/konflux/snapshot.go`): Snapshot validation rules

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for exported functions and types

### CLI Design Principles

1. **Consistency**: Commands should follow similar patterns
2. **Clarity**: Error messages should be helpful and actionable
3. **Flexibility**: Support both interactive and scripted usage
4. **Validation**: Fail fast with clear error messages

### Adding New Commands

1. **Create command structure:**
   ```bash
   mkdir -p cmd/get/newcommand
   ```

2. **Implement command:**
   ```go
   // cmd/get/newcommand/cmd.go
   package newcommand

   import (
       "context"
       "github.com/jordigilh/korn/internal/konflux"
       "github.com/urfave/cli/v3"
   )

   func GetCommand() *cli.Command {
       return &cli.Command{
           Name:    "newcommand",
           Usage:   "description of new command",
           Action:  commandAction,
       }
   }

   func commandAction(ctx context.Context, cmd *cli.Command) error {
       // Implementation
       return nil
   }
   ```

3. **Register command in parent:**
   ```go
   // cmd/get/cmd.go
   func GetCommand() *cli.Command {
       return &cli.Command{
           Commands: []*cli.Command{
               // ... existing commands
               newcommand.GetCommand(),
           },
       }
   }
   ```

4. **Add tests:**
   ```go
   // cmd/get/newcommand/cmd_test.go
   package newcommand_test

   import (
       "testing"
       // test implementation
   )
   ```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/konflux/...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...
```

### Test Structure

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test interactions with Kubernetes APIs
- **CLI Tests**: Test command-line interface behavior

### Writing Tests

1. **Use table-driven tests** for multiple scenarios:
   ```go
   func TestValidateSnapshot(t *testing.T) {
       tests := []struct {
           name     string
           snapshot applicationapiv1alpha1.Snapshot
           want     bool
           wantErr  bool
       }{
           {
               name: "valid snapshot",
               snapshot: validSnapshot,
               want: true,
               wantErr: false,
           },
           // ... more test cases
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got, err := ValidateSnapshot(tt.snapshot)
               if (err != nil) != tt.wantErr {
                   t.Errorf("ValidateSnapshot() error = %v, wantErr %v", err, tt.wantErr)
                   return
               }
               if got != tt.want {
                   t.Errorf("ValidateSnapshot() = %v, want %v", got, tt.want)
               }
           })
       }
   }
   ```

2. **Mock external dependencies** when testing:
   ```go
   type mockKubeClient struct {
       client.Client
       // mock implementation
   }
   ```

3. **Test error conditions** as well as success cases

## Documentation

### Updating Documentation

When making changes:

1. **Update command help text** in the command implementation
2. **Update README** if adding new features
3. **Update docs/** files for detailed documentation
4. **Add examples** to relevant documentation files

### Documentation Structure

- `README.md`: Main project documentation and quick start
- `docs/onboarding.md`: Detailed onboarding process
- `docs/commands.md`: Complete command reference
- `docs/validation-rules.md`: Validation logic explanation
- `docs/examples.md`: Practical examples and workflows
- `docs/contributing.md`: This file

## Submitting Changes

### Pull Request Process

1. **Ensure tests pass:**
   ```bash
   make test
   ```

2. **Update documentation** if needed

3. **Write clear commit messages:**
   ```
   Add snapshot validation for FBC applications

   - Implement basic image existence check
   - Add tests for FBC validation flow
   - Update documentation with FBC examples
   ```

4. **Submit PR with description:**
   - Explain what the change does
   - Why it's needed
   - How to test it
   - Link to related issues

### Review Process

1. **Automated checks** must pass (tests, linting)
2. **Code review** by maintainers
3. **Documentation review** if applicable
4. **Manual testing** for significant changes

## Release Process

### Version Management

- Follow [Semantic Versioning](https://semver.org/)
- Update version in relevant files
- Tag releases in Git

### Release Checklist

1. Update documentation
2. Run full test suite
3. Build and test binary
4. Create release notes
5. Tag version
6. Create GitHub release

## Getting Help

### Community

- **Issues**: Report bugs and request features on GitHub
- **Discussions**: Ask questions in GitHub Discussions
- **Documentation**: Check existing documentation first

### Debugging

1. **Enable debug logging:**
   ```bash
   export LOGRUS_LEVEL=debug
   korn get application
   ```

2. **Check Kubernetes access:**
   ```bash
   kubectl get applications
   oc get applications
   ```

3. **Verify Konflux setup:**
   ```bash
   kubectl get applications,components,snapshots,releases
   ```

## Common Issues

### Development Problems

**Build failures:**
```bash
# Clean and rebuild
make clean
make build
```

**Test failures:**
```bash
# Run specific failing test
go test -v ./internal/konflux/ -run TestSpecificFunction
```

**Import issues:**
```bash
# Update dependencies
go mod tidy
go mod download
```

### Contributing Issues

**PR conflicts:**
```bash
# Rebase on main
git fetch upstream
git rebase upstream/main
```

**Test environment:**
```bash
# Set up test environment variables
export KUBECONFIG=/path/to/kubeconfig
export NAMESPACE=test-namespace
```

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Follow project guidelines

## AI-Generated Content Notice

Contributors should be aware that portions of this codebase have been generated or enhanced using AI tools (primarily Cursor's AI capabilities):

### AI-Generated Components:
- **Unit Tests**: Many test files in `*_test.go` contain AI-generated test cases and mock implementations
- **Documentation**: Parts of the `docs/` directory, including examples and command references, were created with AI assistance
- **Code Comments**: Some inline documentation and function comments were generated or improved using AI
- **Refactoring**: Various code improvements and structural changes were implemented with AI guidance

### For Contributors:
- When reviewing AI-generated code, pay special attention to logic correctness and edge cases
- Feel free to refactor or improve AI-generated content as needed
- All AI-generated content should be treated as any other code contribution - subject to review and testing
- If you use AI tools for your contributions, please ensure the generated code is thoroughly tested and reviewed

## License

By contributing to Korn, you agree that your contributions will be licensed under the same license as the project.