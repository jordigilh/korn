🎉 Introducing Korn
Korn is an opinionated CLI application designed to simplify the release of operators in Konflux by extracting the arduous tasks necessary to ensure successful releases.

🚀 Core Features
• Command Suite: get, create, waitfor with full resource management
• Application lifecycle management and component validation
• Snapshot processing and release orchestration
• Container image validation with version consistency checks
• Git repository integration and Kubernetes cluster support

🔧 Technical Features
• Cross-platform support: Linux (amd64, arm64), macOS (arm64)
• Version management: Git tag-based versioning with --version flag
• Flexible configuration: Kubeconfig and namespace support
• Debug mode and shell completion support

🏗️ Build & Development
• GitHub Actions CI/CD with automated testing and releases
• Comprehensive Ginkgo-based unit tests (>72.9% coverage)
• Cross-compilation with standard binary naming
• Development tools: linting, formatting, code generation

📦 Dependencies
• Konflux APIs: application-api, release-service integration
• Container runtime: Podman v5 support
• Kubernetes: client-go for cluster communication
• CLI framework: urfave/cli v3

🚦 Getting Started
1. Download binary for your platform from releases
2. Configure: korn --kubeconfig ~/.kube/config --namespace my-namespace
3. Explore: korn help
4. Check version: korn --version

First stable release - ready for production use in Konflux operator release workflows.