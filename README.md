# Korn
Korn: an opinionated Konflux Operator Release application

# Building the binary

To build the binary, just run the `make build` target and copy the binary in the `output/` directory to somewhere in your path.

# Motivation

Releasing an operator with Konflux generally translates into a bespoke process involving a set of unique checks and validations that are intertwined with Konflux's own domain constructs (snapshots, releases, release plan and release plan admissions). As such, Konflux only enforces its validations on what's defined by the Enterprise Contract Plan and the RPA's rules, but it is left to the operator's release team to ensure that the release artifacts are consistent, for instance that the image pullspec in the snapshots match the value in the bundle's reference images. This leaves up to the release engineer to define what validations or processes to set in place in order to guarantee the success of a release. In the context of releasing an Operator, the following validations apply in principle:

* The Snapshot object needs to have been created from a push event.
* The Snapshot object needs to be successful.
* The component images referenced in the snapshot exist.
* The related images in the CSV contained in the bundle container image referenced in the snapshot need to match the image references in the snapshot's spec. Failing this validation translates into image specs being pushed to production by the Konflux's release pipeline and the deployment of the operator failing to pull the correct ones due to digest mismatch.
* All version labels found in each of the compoment's container image match.
* Populate the `releaseNotes` in the Release manifest.

Any snapshot that fails any validation cannot be used for a release.

Korn's purpose is to simplify the process of releasing an operator in Konflux by implementing these validations.

# Using korn to release an Operator with Konflux

Korn provides a set of subcommands to operate the Konflux objects with the intent of simplifying the process of releasing the operator's artifacts. In short, it has functionality to read `snapshots`,`applications`,`components` and `releases`. The later also providing a create subcommand. It is in the `get snapshot` and `create release` subcommands where korn executes the series of validations previously mentioned to identify the latest snapshot candidate that passes all validations. The only command you need to run to trigger a release is the `create release -app <app_name> -env <environment_name>`. Example:

```
korn create release -app operator-1-0 -environment staging -releaseNotes releaseNotes-1.0.1.yaml
```

This command will create the release object in the cluster using the latest snapshot candidate for the `operator-1-1` application and using the `staging` RPA associated to the application's Release Plan for that environment, and embed the contents of the `releaseNotes-1.0.1.yaml` file in the `Release` object, for instance [this example](test-data/releaseNotes.rhba) for a bug release and [this one](test-data/releaseNotes.rhsa) for a security release. For more information on the structure of a `releaseNote`, follow this [link](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html#release).

It is possible to retrieve the latest candidate for an application before proceeding to create the release object, as well as generate the `JSON` or `YAML` manifest for the release object instead of creating it. The list of options for creating a release can be retrieved by running the following command:

```
korn create release -h
```


# Onboarding an Operator

Korn uses labels in the existing Konflux's objects in order to identify types of applications, components and which environment belongs to a given ReleasePlan. To onboard your operator into korn, you will have to run the following commands.

## Create the application type label for the `operator` and `fbc` applications:

Konflux's recommends creating one application per OCP version for the FBC (File Based Catalog) and another one for each version of your operator's (controller, console-plugin, must-gather, etc...). In a seasoned operator, you might find a tenant in Konflux such as this:
```
$> oc get application
operator-1-0
operator-1-1
operator-1-2
operator-2-0
fbc-v4-15
fbc-v4-16
fbc-v4-17
fbc-v4-18
fbc-v4-19
```

In order for korn to distinguish the two types of applications listed here (`fbc` and `operator`), each of these applications must be labeled with the following label:

```
korn.redhat.io/application
```

Example:
```
$>oc label application operator-1-0 korn.redhat.io/application=operator
```

And for each FBC application
```
$>oc label application fbc-v4-15 korn.redhat.io/application=fbc
```

Korn leverages on these labels to perform the validations required for releasing a snapshot accordingly. The validations required for releasing an FBC snapshot are not the same for an operator. After you've labeled all applications, you can test the output of running `korn get applications`:

```
$> korn get application
NAME           TYPE       AGE
fbc-v4-15      fbc        59d
fbc-v4-16      fbc        59d
fbc-v4-17      fbc        59d
fbc-v4-18      fbc        59d
fbc-v4-19      fbc        59d
operator-1-0   operator   66d
operator-1-1   operator   66d
operator-1-2   operator   66d
operator-2-0   operator   66d
```

## Create the component type label for the `bundle` component type in the `operator` application:
The `bundle` component in the operator encapsulates the container image where the CSV and other necessary manifests are found. It represents the container image created by the `make bundle-build` make target that is executed using the scaffolding generated by the `operator-sdk` toolkit. Korn's validations for releasing the operator's components are strongly linked to the bundle. And so it needs to be able to identify which one of the components in the `operator` application type contains the `bundle`.

The label has the following format:
```
korn.redhat.io/component
```

Example:
```
$> oc get components
oc get components
NAME                            AGE   STATUS   REASON   TYPE

operator-bundle-1-0             66d
...
operator-bundle-1-1             66d
operator-bundle-1-2             66d
operator-bundle-2-0             66d
fbc-v4-18                       59d
...
...
```

Set the label for each of the `operator-bundle-?-?` components like the example below:
```
$> oc label component operator-bundle-1-0 korn.redhat.io/component=bundle
```

The `korn get component` command should highlight the `bundle` type for the `operator-bundle-?-?` components:

```
$> korn get components -app operator-1-0
NAME                            TYPE     BUNDLE LABEL                AGE
console-plugin-1-0                       console-plugin              67d
controller-rhel9-operator-1-0            controller-rhel9-operator   67d
operator-bundle-1-0             bundle                               67d
```

## Label the ReleasePlan objects for `staging` and `production`
Similarly to what you did previously, you will need to label the `ReleasePlan` objects in your namespace with a label that identifies them whether they are `staging` or `production` plans. Korn uses this label to reference the plan in the `Release` manifest based on the target environment: `staging` or `production`.

Assuming you have 2 plans already for each application:
```
$> oc get releaseplan
NAME                                  APPLICATION    TARGET
fbc-v4-15-release-as-staging-fbc      fbc-v4-15      rhtap-releng-tenant
...
fbc-v4-15-release-as-production-fbc   fbc-v4-15      rhtap-releng-tenant
operator-staging-1-0                  operator-1-0   rhtap-releng-tenant
...
operator-production-1-0               operator-1-0   rhtap-releng-tenant
...
```

Set the label for each plan accordingly for `staging` or `production`:

```
$> oc label releaseplan fbc-v4-15-release-as-staging-fbc korn.redhat.io/environment=staging
releaseplan.appstudio.redhat.com/fbc-v4-15-release-as-staging-fbc labeled
...
$> oc label releaseplan fbc-v4-15-release-as-production-fbc korn.redhat.io/environment=production
releaseplan.appstudio.redhat.com/fbc-v4-15-release-as-production-fbc labeled
...
$> oc label releaseplan operator-staging-1-0 korn.redhat.io/environment=staging
releaseplan.appstudio.redhat.com/operator-staging-1-0 labeled
...
$> oc label releaseplan operator-production-1-0 korn.redhat.io/environment=production
releaseplan.appstudio.redhat.com/operator-production-1-0 labeled
```

Korn should be then able to identify each ReleasePlan accordingly:

```
$> korn get releaseplans -app operator-1-0
NAME                      APPLICATION    ENVIRONMENT      RELEASE PLAN ADMISSION                                 ACTIVE   AGE
operator-staging-1-0      operator-1-0   staging       	  rhtap-releng-tenant/my-operator-staging-1-0   		 true     66d
operator-production-1-0   operator-1-0   production       rhtap-releng-tenant/my-operator-prod-1-0   			 true     66d
```

Make sure these changes are persisted across updates by updating your source manifests in the konflux repository

## Update the bundle's containerfile

So far we've been adding labels to existing objects, which is not an intrusive operation and does not affect your operator's code structure. The next step, however, is going to change that. In order for Korn to be able to
ensure that the image pullspecs referenced in the bundle match the ones in the snapshot, it needs to be able to match the name of the component in the snapshot's spec in the bundle's container image. The way it is done is by performing a lookup for a label in the bundle's container image that matches the component's name. The label's value equals to the image pullspec referenced in the bundle's manifests. Example:

```
FROM scratch

ARG VERSION=1.0

LABEL controller-rhel9-operator="registry.stage.redhat.io/my-operator-tech-preview/my-rhel9-operator@sha256:6b33780302d877c80f3775673aed629975e6bebb8a8bd3499a9789bd44d04861"
LABEL console-plugin="registry.stage.redhat.io/my-operator-tech-preview/my-console-plugin-rhel9@sha256:723276c6a1441d6b0d13b674b905385deec0291ac458260a569222b5612f73c4"

COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
COPY bundle/tests/scorecard /tests/scorecard/

COPY LICENSE /licenses/licenses
...
```

The label `controller-rhel9-operator` defines the location of the pullspec for the controller component. Likewise for the `console-plugin` and so on for as many components as referenced in the snapshot. Korn expects a label in each component that is part of an operator type application that defines the label name to use when looking up the image digest in the bundle's container image. The label has the following format:

```
korn.redhat.io/bundle-label
```

In this current example, we will set the labels for the `controller-rhel9-operator-1-0` and `console-plugin-1-0` components:

```
$> oc label component console-plugin-1-0 korn.redhat.io/bundle-label=console-plugin
component.appstudio.redhat.com/console-plugin-1-0 labeled
$> oc label component controller-rhel9-operator-1-0 korn.redhat.io/bundle-label=controller-rhel9-operator
component.appstudio.redhat.com/controller-rhel9-operator-1-0 labeled
```

You will also need to ensure that your `bundle.Dockefile` contains these 2 labels from now on, otherwise `korn` won't be able to validate the snapshot's candidacy for release. This is where you can get creative and figure out a way to do it automatically, such as with nudges or scripts that generate the bundle containerfile on demand prior to triggering the build task in the konflux pipeline. Nudges should be your best bet, they will trigger PRs that update the pullspec reference in the bundle.Dockerfile. Unless you end up with PR merge conflicts every time multiple nudges are triggered in parallel due to a change in one artifact in your repository, in which case a script that generates the containerfile is the way to go, but it requires further changes to your repository and build pipeline. Listing the components with the `korn` cli should render the value of the bundle labels for each component:

```
$> oc get component -app operator-1-0
NAME                            TYPE     BUNDLE LABEL                AGE
console-plugin-1-0                       console-plugin              67d
controller-rhel9-operator-1-0            controller-rhel9-operator   67d
operator-bundle-1-0             bundle                               67d
```

The bundle does not need a label, since its image pullspec is not referenced inside the bundle's manifests. For simplicity purposes, it's best to leave out any version from the label's name to avoid having to update the references in the `bundle.Dockerfile` every time a new version is created.


## FBC applications
Following Konflux's general advice, FBC applications are expected to contain only one single component mapping to a specific OCP version. The process of releasing an FBC application is much simpler than releasing operator images since the catalog manifest is manually updated with the new reference to the bundle's image pullspec. Korn does not assist in this area, since the task of updating the catalog also entails creating a PR and it's best to manually perform the action at this point.

Like with the operator's application, `korn` supports the ability to automatically select the latest valid snapshot for the FBC application. The validation will check that the snapshot was successful and that the container image exists.
