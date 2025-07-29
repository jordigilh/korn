# Examples and Workflows

This guide provides practical examples and workflows for common Korn operations.

## Complete Setup Workflow

### Initial Onboarding

```bash
# 1. Label applications by type
oc label application operator-1-0 korn.redhat.io/application=operator
oc label application fbc-v4-15 korn.redhat.io/application=fbc

# 2. Label bundle components
oc label component operator-bundle-1-0 korn.redhat.io/component=bundle

# 3. Set up bundle-component mapping
oc label component controller-rhel9-operator-1-0 korn.redhat.io/bundle-label=controller-rhel9-operator
oc label component console-plugin-1-0 korn.redhat.io/bundle-label=console-plugin

# 4. Label release plans by environment
oc label releaseplan operator-staging-1-0 korn.redhat.io/environment=staging
oc label releaseplan operator-production-1-0 korn.redhat.io/environment=production

# 5. Verify setup
korn get application
korn get component --app operator-1-0
korn get releaseplan --app operator-1-0
```

### Bundle Dockerfile Updates

Ensure your `bundle.Dockerfile` includes the component image labels:

```dockerfile
FROM scratch

ARG VERSION=1.0

# Component image labels matching korn.redhat.io/bundle-label values
LABEL controller-rhel9-operator="registry.stage.redhat.io/my-operator/controller@sha256:abc123..."
LABEL console-plugin="registry.stage.redhat.io/my-operator/console-plugin@sha256:def456..."

COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
COPY bundle/tests/scorecard /tests/scorecard/
COPY LICENSE /licenses/licenses
```

## Release Workflows

### Standard Release Workflow

```bash
# 1. Check available applications
korn get application

# 2. Examine application components and their status
korn get component --app operator-1-0

# 3. Review available release plans
korn get releaseplan --app operator-1-0

# 4. Get the latest valid snapshot candidate and capture its name
SNAPSHOT=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')
echo "Using snapshot: $SNAPSHOT"

# 5. Create staging release with captured snapshot
STAGING_RELEASE=$(korn create release --app operator-1-0 --environment staging --snapshot $SNAPSHOT --wait=false | grep "Release created" | awk '{print $3}')

# 6. Wait for staging release completion (optional)
korn waitfor release $STAGING_RELEASE

# 7. After staging validation, promote to production using same snapshot
korn create release --app operator-1-0 --environment production --snapshot $SNAPSHOT
```

### Release with Specific Snapshot

```bash
# 1. List recent snapshots
korn get snapshot --app operator-1-0

# 2. Choose specific snapshot
SNAPSHOT="snapshot-sample-xyz123"

# 3. Create release with chosen snapshot
korn create release \
  --app operator-1-0 \
  --environment staging \
  --snapshot $SNAPSHOT \
  --releaseNotes release-notes.yaml
```

### Release with Release Notes

```bash
# 1. Prepare release notes file (release-notes.yaml)
cat > release-notes.yaml << EOF
type: "RHBA"  # Bug release
issues:
  - "Fixed authentication timeout issue"
  - "Improved error handling in controller"
reference:
  - "https://issues.redhat.com/browse/EXAMPLE-123"
EOF

# 2. Create release with notes
korn create release \
  --app operator-1-0 \
  --environment production \
  --releaseNotes release-notes.yaml
```

### Force Release (Retry Failed Release)

```bash
# When previous release failed and you want to retry
korn create release \
  --app operator-1-0 \
  --environment staging \
  --force \
  --releaseNotes updated-release-notes.yaml
```

## Validation and Debugging Workflows

### Pre-Release Validation

```bash
# 1. Check application setup
korn get application

# 2. Verify all components are properly labeled
korn get component --app operator-1-0

# 3. Check latest candidate snapshot
korn get snapshot --app operator-1-0 --candidate

# 4. Review snapshot details
CANDIDATE=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')
korn get snapshot $CANDIDATE

# 5. Test release creation (dry run)
korn create release \
  --app operator-1-0 \
  --environment staging \
  --snapshot $CANDIDATE \
  --dryrun \
  --output yaml
```

### Debugging Failed Validations

```bash
# 1. Check all snapshots for the application
korn get snapshot --app operator-1-0

# 2. Look for specific snapshot by commit
korn get snapshot --app operator-1-0 --sha abc1234def5678

# 3. Check snapshot by version
korn get snapshot --app operator-1-0 --version v1.0.15

# 4. Verify component configuration
kubectl get components -l appstudio.openshift.io/application=operator-1-0 -o yaml

# 5. Check bundle component specifically
kubectl get component operator-bundle-1-0 -o yaml | grep -A10 labels

# 6. Review recent releases
korn get release --app operator-1-0
```

### Manual Image Validation

```bash
# 1. Get bundle component details
BUNDLE_COMPONENT=$(korn get component --app operator-1-0 | grep bundle | awk '{print $1}')

# 2. Get latest snapshot
SNAPSHOT=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}')

# 3. Extract bundle image from snapshot
kubectl get snapshot $SNAPSHOT -o jsonpath='{.spec.components[?(@.name=="'$BUNDLE_COMPONENT'")].containerImage}'

# 4. Inspect bundle labels manually
BUNDLE_IMAGE="registry.../bundle@sha256:..."
podman inspect $BUNDLE_IMAGE | jq '.config.Labels'
```

## Advanced Workflows

### Multi-Environment Release Pipeline

```bash
#!/bin/bash
# multi-env-release.sh

APP_NAME="operator-1-0"
RELEASE_NOTES="release-notes.yaml"

# 1. Get latest candidate
echo "Finding latest candidate snapshot..."
SNAPSHOT=$(korn get snapshot --app $APP_NAME --candidate | tail -n 1 | awk '{print $1}')
echo "Using snapshot: $SNAPSHOT"

# 2. Create staging release
echo "Creating staging release..."
STAGING_RELEASE=$(korn create release \
  --app $APP_NAME \
  --environment staging \
  --snapshot $SNAPSHOT \
  --releaseNotes $RELEASE_NOTES \
  --wait=false | grep "Release created" | awk '{print $3}')

echo "Staging release: $STAGING_RELEASE"

# 3. Wait for staging completion
echo "Waiting for staging release to complete..."
korn waitfor release $STAGING_RELEASE --timeout 120

# 4. Create production release
echo "Creating production release..."
PROD_RELEASE=$(korn create release \
  --app $APP_NAME \
  --environment production \
  --snapshot $SNAPSHOT \
  --releaseNotes $RELEASE_NOTES \
  --wait=false | grep "Release created" | awk '{print $3}')

echo "Production release: $PROD_RELEASE"

# 5. Wait for production completion
echo "Waiting for production release to complete..."
korn waitfor release $PROD_RELEASE --timeout 180

echo "Release pipeline completed successfully!"
```

### Version-Based Release Workflow

Release a specific version by finding snapshots that match the version tag.

> **Required:** Your git repository must have a `VERSION.txt` file in the root containing the version (e.g., `1.0.15`).

```bash
#!/bin/bash
# version-release.sh

APP_NAME="operator-1-0"
VERSION="v1.0.15"

# 1. Find snapshot for specific version
echo "Finding snapshot for version $VERSION..."
SNAPSHOT=$(korn get snapshot --app $APP_NAME --version $VERSION | tail -n 1 | awk '{print $1}')

if [ -z "$SNAPSHOT" ]; then
  echo "No snapshot found for version $VERSION"
  exit 1
fi

echo "Found snapshot: $SNAPSHOT"

# 2. Create release notes based on version
cat > version-release-notes.yaml << EOF
type: "RHEA"  # Enhancement release
issues:
  - "Version $VERSION release"
  - "Updated components to $VERSION"
reference:
  - "https://github.com/myorg/operator/releases/tag/$VERSION"
EOF

# 3. Create production release
korn create release \
  --app $APP_NAME \
  --environment production \
  --snapshot $SNAPSHOT \
  --releaseNotes version-release-notes.yaml

echo "Version $VERSION release created successfully"
```

### Automated Bundle Validation

```bash
#!/bin/bash
# validate-bundle.sh

APP_NAME="operator-1-0"

# 1. Get bundle component
BUNDLE=$(korn get component --app $APP_NAME | grep bundle | awk '{print $1}')

# 2. Get latest snapshot
SNAPSHOT=$(korn get snapshot --app $APP_NAME --candidate | tail -n 1 | awk '{print $1}')

# 3. Extract bundle image
BUNDLE_IMAGE=$(kubectl get snapshot $SNAPSHOT -o jsonpath='{.spec.components[?(@.name=="'$BUNDLE'")].containerImage}')

echo "Validating bundle: $BUNDLE_IMAGE"

# 4. Check bundle labels
podman inspect $BUNDLE_IMAGE --format '{{range $key, $value := .Config.Labels}}{{if and (ne $key "version") (ne $key "name")}}{{$key}}={{$value}}{{printf "\n"}}{{end}}{{end}}' > bundle-labels.txt

# 5. Check snapshot components
kubectl get snapshot $SNAPSHOT -o jsonpath='{.spec.components[*].name}' | tr ' ' '\n' > snapshot-components.txt

# 6. Compare bundle labels with snapshot components
echo "Bundle Labels:"
cat bundle-labels.txt

echo -e "\nSnapshot Components:"
cat snapshot-components.txt

# 7. Verify each component has matching bundle label
while read component; do
  if [ "$component" != "$BUNDLE" ]; then
    BUNDLE_LABEL=$(kubectl get component $component -o jsonpath='{.metadata.labels.korn\.redhat\.io/bundle-label}')
    if grep -q "^$BUNDLE_LABEL=" bundle-labels.txt; then
      echo "✅ Component $component has matching bundle label: $BUNDLE_LABEL"
    else
      echo "❌ Component $component missing bundle label: $BUNDLE_LABEL"
    fi
  fi
done < snapshot-components.txt

# Cleanup
rm bundle-labels.txt snapshot-components.txt
```

## FBC Release Workflow

```bash
# FBC applications have simpler workflows

# 1. Verify FBC application setup
korn get application | grep fbc

# 2. Check FBC snapshot and capture its name
SNAPSHOT=$(korn get snapshot --app fbc-v4-15 --candidate | tail -n 1 | awk '{print $1}')

# 3. Create FBC release (no bundle validation)
korn create release --app fbc-v4-15 --environment staging --snapshot $SNAPSHOT

# 4. Manual catalog update (outside Korn)
# Update catalog files and create PR for catalog updates
```

## Troubleshooting Common Issues

### Issue: No Valid Candidate Found

```bash
# Check all snapshots
korn get snapshot --app operator-1-0

# Look for failed snapshots
kubectl get snapshots -l appstudio.openshift.io/application=operator-1-0 \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="AppStudioTestSucceeded")].reason}{"\n"}{end}'

# Check last successful release
korn get release --app operator-1-0 | head -5
```

### Issue: Bundle Validation Failures

```bash
# Check bundle component labels
kubectl get component operator-bundle-1-0 -o yaml | grep -A5 labels

# Verify bundle Dockerfile has required labels
BUNDLE_IMAGE=$(korn get snapshot --app operator-1-0 --candidate | tail -n 1 | awk '{print $1}' | xargs kubectl get snapshot -o jsonpath='{.spec.components[?(@.name=="operator-bundle-1-0")].containerImage}')
podman inspect $BUNDLE_IMAGE | jq '.config.Labels'

# Check component bundle-label annotations
kubectl get components -l appstudio.openshift.io/application=operator-1-0 \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.labels.korn\.redhat\.io/bundle-label}{"\n"}{end}'
```

### Issue: Environment Targeting Problems

```bash
# Check release plan labels
kubectl get releaseplans -l appstudio.openshift.io/application=operator-1-0 \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.labels.korn\.redhat\.io/environment}{"\n"}{end}'

# Verify active release plan admissions
kubectl get releaseplans -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.releasePlanAdmission.active}{"\n"}{end}'
```

## Advanced Snapshot Selection

### Version-Specific Candidate Selection

Use `--version` and `--candidate` together to find the best candidate from a specific version.

> **Prerequisites:** Your git repository must have a `VERSION.txt` file in the root directory containing a valid semantic version (e.g., `1.0.15`) for each commit that you want to be discoverable by version filtering.

**Example VERSION.txt file:**
```
1.0.15
```

```bash
# Get all snapshots for a specific version (useful for debugging)
korn get snapshot --app operator-1-0 --version v1.0.15

# Get the best release candidate from a specific version
korn get snapshot --app operator-1-0 --version v1.0.15 --candidate

# Workflow: Release a specific version
VERSION="v1.0.15"
SNAPSHOT=$(korn get snapshot --app operator-1-0 --version $VERSION --candidate | tail -n 1 | awk '{print $1}')
korn create release --app operator-1-0 --environment staging --snapshot $SNAPSHOT

# Compare candidates across versions
echo "Candidates for v1.0.15:"
korn get snapshot --app operator-1-0 --version v1.0.15 --candidate
echo "Candidates for v1.1.0:"
korn get snapshot --app operator-1-0 --version v1.1.0 --candidate
```

> **Use Case**: This is particularly useful when you need to create a release from a specific version while ensuring you get the most suitable candidate snapshot that hasn't been used in previous releases.