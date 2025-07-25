# Validation Rules

Korn performs comprehensive validations to ensure that snapshots are suitable for release. The validation rules differ based on application type.

## Overview

Korn's validation process ensures that:
- Snapshots contain properly tested and validated images
- Image references in bundle CSVs match actual snapshot images
- Version labels are consistent across all components
- All components are accessible and deployable

## Operator Applications

Operator applications undergo rigorous validation due to their complex multi-component architecture.

### Snapshot Validation

**✅ Snapshot created from push event**
- Verifies the snapshot was triggered by a code push (not manual creation)
- Ensures proper CI/CD workflow was followed

**✅ Snapshot marked as successful**
- Checks that `AppStudioTestSucceeded` condition is `Finished`
- Confirms all tests passed before considering for release

**✅ All component images exist and are accessible**
- Validates each component image can be pulled
- Ensures container registry accessibility
- Verifies image digests are valid

### Bundle Validation

**✅ Bundle CSV references match snapshot image specs**
- Compares image references in bundle's CSV with snapshot component images
- Validates SHA256 digests match between bundle labels and snapshot specs
- Prevents deployment failures due to image reference mismatches

**Example validation:**
```bash
# Bundle Dockerfile label
LABEL controller-rhel9-operator="registry.../controller@sha256:abc123..."

# Must match snapshot component
snapshot.spec.components[0].containerImage = "registry.../controller@sha256:abc123..."
```

**✅ Version labels consistent across all components**
- Ensures all component images have matching version labels
- Prevents mixed-version releases
- Validates semantic versioning consistency

### Bundle-Component Mapping

Korn validates that:
1. Each component has a corresponding label in the bundle image
2. The label value matches the component's image digest in the snapshot
3. All components referenced in the CSV are present in the snapshot

## FBC Applications

FBC (File Based Catalog) applications have simpler validation requirements.

**✅ Snapshot marked as successful**
- Checks that `AppStudioTestSucceeded` condition is `Finished`
- Confirms catalog image passed validation

**✅ Container image exists and is accessible**
- Validates the catalog image can be pulled
- Ensures container registry accessibility

> **Note:** FBC releases require manual catalog updates via PR - Korn assists with snapshot validation only.

## Validation Workflow

### 1. Discovery Phase
```bash
# Korn identifies application type
kubectl get application operator-1-0 -o jsonpath='{.metadata.labels.korn\.redhat\.io/application}'

# Finds bundle component for operators
kubectl get components -l korn.redhat.io/component=bundle
```

### 2. Snapshot Assessment
- Lists all snapshots for the application
- Filters by successful test status
- Orders by creation timestamp (newest first)

### 3. Candidacy Validation
For each potential snapshot:
- Checks if it's newer than the last successful release
- Validates all validation rules for the application type
- Returns first valid candidate found

### 4. Bundle Analysis (Operators Only)
- Pulls bundle container image
- Extracts and parses CSV manifests
- Compares component references with snapshot specs
- Validates version consistency

## Validation Failures

### Common Failure Scenarios

**Snapshot Test Failures:**
```
snapshot snapshot-xyz123 has not finished running yet, discarding
```
- Snapshot tests are still running or failed
- Wait for tests to complete or fix test failures

**Image Reference Mismatches:**
```
component controller pullspec mismatch in bundle operator-bundle-1-0, snapshot is not a candidate for release
```
- Bundle CSV references different image digest than snapshot
- Update bundle labels or rebuild components

**Version Inconsistencies:**
```
component controller and bundle operator-bundle-1-0 version mismatch: component has v1.0.1 and bundle has v1.0.0
```
- Component image has different version label than bundle
- Ensure all images use consistent version labeling

**Missing Bundle Labels:**
```
missing label controller-rhel9-operator for component controller in bundle container image
```
- Bundle Dockerfile missing required component label
- Add missing LABEL directive to bundle.Dockerfile

**Component Not Found:**
```
component reference controller-rhel9-operator in snapshot snapshot-xyz123 not found
```
- Snapshot missing expected component
- Verify component build completed successfully

### Debugging Validation Issues

**Check snapshot status:**
```bash
korn get snapshot snapshot-xyz123
```

**Verify component images:**
```bash
korn get component --app operator-1-0
```

**Test manual validation:**
```bash
# Check if image exists
podman pull registry.../controller@sha256:abc123...

# Inspect bundle labels
podman inspect registry.../bundle@sha256:def456... | jq '.config.Labels'
```

**Review application labels:**
```bash
kubectl get application operator-1-0 -o yaml | grep -A5 labels
kubectl get components -l appstudio.openshift.io/application=operator-1-0 -o yaml | grep -A10 labels
```

## Best Practices

### For Operators

1. **Consistent Versioning**: Use the same version label across all component images
2. **Automated Bundle Updates**: Use nudges or scripts to keep bundle labels synchronized
3. **Version Validation**: Implement CI checks to verify bundle-component consistency
4. **Test Automation**: Ensure comprehensive test coverage for all components

### For FBC Applications

1. **Minimal Validation**: Focus on image accessibility and basic functionality
2. **Manual Updates**: Plan for manual catalog updates after successful releases
3. **Version Control**: Track catalog changes through proper git workflows

### General Guidelines

1. **Monitor Snapshots**: Regularly check snapshot status and validation results
2. **Label Management**: Keep Korn labels up-to-date across all resources
3. **Release Notes**: Maintain clear release documentation
4. **Environment Progression**: Always validate in staging before production releases