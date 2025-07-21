# Operator Onboarding

To use Korn with your operator, you need to label existing Konflux resources appropriately.

## Why Labels Are Required

Korn operates on existing Konflux resources (Applications, Components, ReleasePlans) that don't inherently contain information about their role in the operator release process. While Konflux provides the infrastructure and enforces its own validations through Enterprise Contract Plans, it doesn't distinguish between different types of applications or understand operator-specific concepts like bundle components.

**Korn uses labels as a discovery and classification mechanism** to bridge this gap and enable intelligent automation of operator releases.

### Application Type Classification (`korn.redhat.io/application`)

Konflux treats all applications equally, but operator and FBC applications require fundamentally different validation strategies. Operator applications contain multiple interdependent components with complex image reference relationships that must be validated against CSV manifests. FBC applications, in contrast, are simpler single-component catalog containers that require only basic image existence checks.

By labeling applications as `operator` or `fbc`, Korn can apply the appropriate validation rules automatically. This enables sophisticated operator-specific validations like CSV parsing and image consistency checks for operator applications, while applying simpler validation logic for FBC applications.

**Example:**
```bash
# Operator application with multiple components requiring CSV validation
oc label application operator-1-0 korn.redhat.io/application=operator

# FBC application with single catalog component requiring basic checks
oc label application fbc-v4-15 korn.redhat.io/application=fbc
```

When Korn processes `operator-1-0`, it performs bundle CSV parsing, image digest validation, and version consistency checks. For `fbc-v4-15`, it only verifies the catalog image exists and is accessible.

### Component Role Identification (`korn.redhat.io/component`)

Within operator applications, multiple components typically exist representing different parts of the operator ecosystem (controller, console-plugin, bundle, must-gather, etc.). However, Korn needs to specifically identify the bundle component since it contains the CSV manifests that define image references for all other components.

The `bundle` label designation allows Korn to locate the correct bundle container image for CSV parsing and validation. Without this identification, Korn would have no way to determine which component contains the critical metadata needed for validation.

**Example:**
```bash
# Multiple components in operator-1-0 application
oc get components
NAME                            AGE
controller-rhel9-operator-1-0   67d
console-plugin-1-0              67d
operator-bundle-1-0             67d
must-gather-1-0                 67d

# Label the bundle component specifically
oc label component operator-bundle-1-0 korn.redhat.io/component=bundle
```

Now when Korn searches for CSV manifests, it knows to pull and inspect the `operator-bundle-1-0` container image, ignoring the other components that don't contain bundle metadata.

### Bundle-Component Mapping (`korn.redhat.io/bundle-label`)

One of Korn's most critical validations involves verifying that image references declared in the bundle's CSV match the actual component images present in the snapshot. This prevents deployment failures where the bundle references images with different digests than what's actually being released.

Each component specifies which label name to look for in the bundle's container image through the `bundle-label` annotation. This creates a direct mapping between logical component names and the physical labels in the bundle's Dockerfile, enabling automated verification that bundle references match snapshot reality.

**Example:**
```bash
# Components specify their bundle label names
oc label component controller-rhel9-operator-1-0 korn.redhat.io/bundle-label=controller-rhel9-operator
oc label component console-plugin-1-0 korn.redhat.io/bundle-label=console-plugin

# Corresponding labels in bundle.Dockerfile
LABEL controller-rhel9-operator="registry.stage.redhat.io/my-operator/controller@sha256:abc123..."
LABEL console-plugin="registry.stage.redhat.io/my-operator/console-plugin@sha256:def456..."
```

Korn validates that the snapshot contains `controller-rhel9-operator-1-0` with digest `sha256:abc123...` and `console-plugin-1-0` with digest `sha256:def456...`, ensuring bundle and snapshot consistency.

### Environment Targeting (`korn.redhat.io/environment`)

Most applications maintain separate ReleasePlans for different environments (staging, production, development), but users shouldn't need to memorize specific ReleasePlan names or manage environment-to-plan mappings manually.

By labeling ReleasePlans with their target environment (`staging` or `production`), Korn can automatically select the appropriate plan based on user intent. This enables simple commands like `korn create release --environment staging` without requiring users to specify exact ReleasePlan names.

**Example:**
```bash
# Multiple ReleasePlans with complex names
oc get releaseplan
NAME                                  APPLICATION    TARGET
operator-release-plan-staging-1-0     operator-1-0   rhtap-releng-tenant
operator-release-plan-production-1-0  operator-1-0   rhtap-releng-tenant

# Label them by environment for easy targeting
oc label releaseplan operator-release-plan-staging-1-0 korn.redhat.io/environment=staging
oc label releaseplan operator-release-plan-production-1-0 korn.redhat.io/environment=production

# Simple command now works
korn create release --app operator-1-0 --environment staging
# Automatically finds and uses operator-release-plan-staging-1-0
```

## Label Schema

| Label | Resource Type | Values | Purpose |
|-------|---------------|--------|---------|
| `korn.redhat.io/application` | Application | `operator`, `fbc` | Determines validation strategy |
| `korn.redhat.io/component` | Component | `bundle` | Identifies bundle components |
| `korn.redhat.io/bundle-label` | Component | `<label-name>` | Maps to bundle Dockerfile labels |
| `korn.redhat.io/environment` | ReleasePlan | `staging`, `production` | Environment targeting |

## Validation Workflow

1. **Discovery**: Korn uses labels to find relevant resources
   ```bash
   # Find operator applications
   kubectl get applications -l korn.redhat.io/application=operator

   # Find bundle components
   kubectl get components -l korn.redhat.io/component=bundle
   ```

2. **Classification**: Different validation rules apply based on labels
   ```
   operator applications → CSV validation, image consistency checks
   fbc applications → basic image existence checks
   ```

3. **Mapping**: Bundle labels connect logical components to physical images
   ```
   component "controller-rhel9-operator" → bundle label "controller-rhel9-operator"
   → LABEL controller-rhel9-operator="registry.../image@sha256:..."
   ```

4. **Targeting**: Environment labels select appropriate ReleasePlans
   ```bash
   korn create release --environment staging
   # → finds ReleasePlan with korn.redhat.io/environment=staging
   ```

## What Breaks Without Labels

**Missing Application Labels:**
- Korn can't distinguish operator from FBC applications
- Wrong validation rules applied (or no validation)
- Releases may succeed but deployments fail

**Missing Component Labels:**
- Can't locate bundle component for CSV parsing
- No image reference validation possible
- Silent inconsistencies between bundle and snapshot

**Missing Bundle-Label Mapping:**
- Can't verify bundle references match snapshot images
- Potential runtime failures when Kubernetes can't pull images
- No early detection of image reference mismatches

**Missing Environment Labels:**
- Can't automatically select correct ReleasePlan
- Users must specify exact ReleasePlan names
- No environment-based workflow automation

## Step-by-Step Setup

### 1. Application Labels

Apply the `korn.redhat.io/application` label to distinguish between operator and FBC applications (see [Application Type Classification](#application-type-classification-kornredhatioApplication) for detailed explanation and examples).

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

Apply component labels to identify bundle components and establish bundle-component mapping (see [Component Role Identification](#component-role-identification-kornredhatioComponent) and [Bundle-Component Mapping](#bundle-component-mapping-kornredhatiotbundle-label) for detailed explanations and examples).

**Verify component labeling:**
```bash
korn get component --app operator-1-0
```

Expected output:
```
NAME                            TYPE     BUNDLE LABEL                AGE
console-plugin-1-0                       console-plugin              67d
controller-rhel9-operator-1-0            controller-rhel9-operator   67d
operator-bundle-1-0             bundle                               67d
```

### 3. Release Plan Labels

Apply environment labels to ReleasePlans for automatic environment targeting (see [Environment Targeting](#environment-targeting-kornredhatioEnvironment) for detailed explanation and examples).

**Verify release plan labeling:**
```bash
korn get releaseplan --app operator-1-0
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