# Contributing to GPU Operator Kyma Module

Thank you for your interest in contributing to the GPU Operator Kyma Module!

## Code of Conduct

This project adheres to the Kyma [Code of Conduct](https://github.com/kyma-project/.github/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Issues

If you find a bug or have a feature request, please open an issue on GitHub:

1. Search existing issues to avoid duplicates
2. Use the issue template if available
3. Provide as much context as possible:
   - Environment details (Kyma version, Kubernetes version, cloud provider)
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs and screenshots

### Submitting Pull Requests

1. **Fork the repository** and create your branch from `main`

2. **Set up your development environment**:
   ```bash
   # Clone your fork
   git clone https://github.com/<your-username>/gpu-operator.git
   cd gpu-operator
   
   # Install dependencies
   go mod download
   ```

3. **Make your changes**:
   - Write clear, concise commit messages
   - Follow Go coding standards and conventions
   - Add tests for new functionality
   - Update documentation as needed

4. **Test your changes**:
   ```bash
   # Run unit tests
   make test
   
   # Run linter
   make lint
   
   # Build the project
   make build
   ```

5. **Generate manifests** if you modified the API:
   ```bash
   make manifests generate
   ```

6. **Create a pull request**:
   - Provide a clear description of the changes
   - Reference any related issues
   - Ensure all CI checks pass

### Development Workflow

#### Prerequisites

- Go 1.23 or later
- kubebuilder 3.x
- kubectl
- Access to a Kubernetes cluster (for integration testing)
- Docker or Podman

#### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires a cluster)
make test-integration
```

#### Running Locally

```bash
# Install CRDs
make install

# Run the controller locally
make run
```

#### Building Images

```bash
# Build and push the controller image
make docker-build docker-push IMG=<your-registry>/gpu-operator:<tag>
```

### Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and concise
- Write descriptive error messages

### Commit Message Guidelines

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Example:
```
feat(controller): add support for custom Helm values

Added support for referencing a ConfigMap containing custom
NVIDIA GPU Operator Helm values, allowing users to override
default configuration.

Closes #123
```

### Documentation

- Update README.md for user-facing changes
- Add inline code comments for complex logic
- Update API documentation for CRD changes
- Include examples for new features

### Testing Guidelines

- Write unit tests for new functions
- Add integration tests for controller logic
- Test error handling and edge cases
- Verify backward compatibility
- Test on different Kubernetes versions if possible

### Release Process

Releases are managed by maintainers. The process includes:

1. Version bump in module-config.yaml
2. Update CHANGELOG.md
3. Create and push a git tag
4. Build and push container images
5. Create module bundle with modulectl
6. Update ModuleTemplate in Control Plane

## Getting Help

- Open a GitHub issue for bugs or feature requests
- Join the Kyma community Slack channel
- Check the [Kyma documentation](https://kyma-project.io/docs/)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
