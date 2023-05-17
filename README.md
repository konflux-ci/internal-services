# Internal services

Internal services is a Kubernetes operator to offer a way to run internal queries from Stonesoup operators.

## Table of contents

- [Running, building and testing the operator](#running-building-and-testing-the-operator)
    - [Running the operator locally](#running-the-operator-locally)
    - [Build and push a new image](#build-and-push-a-new-image)
    - [Running tests](#running-tests)
- [Disabling Webhooks for local development](#disabling-webhooks-for-local-development)
- [Accessing the logs](#accessing-the-logs)
    - [CloudWatch](#cloudwatch)
    - [Splunk](#splunk)
- [Testing the operator using Kind](#testing-the-operator-using-kind)
    - [Installing `kind`](#installing-kind)
    - [Installing `kubectl`](#installing-kubectl)
    - [Creating the clusters](#creating-the-clusters)
        - [`local` cluster terminal](#local-cluster-terminal)
        - [`remote` cluster terminal](#remote-cluster-terminal)
    - [Clone and run the operator](#clone-and-run-the-operator)
        - [Installing the internal-services CR](#installing-the-internal-services-cr)
        - [Running the operator in the cluster](#running-the-operator-in-the-cluster)
    - [Testing an InternalRequest](#testing-an-internalrequest)
        - [Sample Pipeline](#sample-pipeline)
        - [Sample InternalRequest](#sample-internalrequest)
- [Links](#links)

## Running, building and testing the operator

This operator provides a [Makefile](Makefile) to run all the usual development tasks. This file can be used by cloning
the repository and running `make` over any of the provided targets.

### Running the operator locally

When testing locally (eg. a CRC cluster), the command `make run install` can be used to deploy and run the operator. 
If any change has been done in the code, `make manifests generate` should be executed before to generate the new resources
and build the operator.

### Build and push a new image

To build the operator and push a new image to the registry, the following commands can be used: 

```shell
$ make docker-build
$ make docker-push
```

These commands will use the default image and tag. To modify them, new values for `TAG` and `IMG` environment variables
can be passed. For example, to override the tag:

```shell
$ TAG=my-tag make docker-build
$ TAG=my-tag make docker-push
```

Or, in the case the image should be pushed to a different repository:

```shell
$ IMG=quay.io/user/internal-services:my-tag make docker-build
$ IMG=quay.io/user/internal-services:my-tag make docker-push
```

### Running tests

To test the code, simply run `make test`. This command will fetch all the required dependencies and test the code. The
test coverage will be reported at the end, once all the tests have been executed.

## Disabling Webhooks for local development

Webhooks require self-signed certificates to validate the resources. To disable webhooks during local development and
testing, export the `ENABLE_WEBHOOKS` variable setting its value to `false` or prepend it while running the operator
using the following command:

```shell
$ ENABLE_WEBHOOKS=false make run install
```
## Accessing the logs

### CloudWatch

The procedure to request access to CloudWatch is documented in the [Getting Access](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/stonesoup/sop/getting-access.md#gain-view-access-to-appsre-logs-including-logs-from-the-fleet-manager-cluster-and-argocd) page.
Make sure MFA is enabled to your account, otherwise it will not be possible to access the logs.

Once in the CloudWatch web console you can search for `stonesoup-int-svc` in the log group search box.

### Splunk

First access the [Splunk & app-sre](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/app-sre/splunk.md#getting-support-working-with-it) page and complete the "Access to Monitoring Platform" request.

Once you have the access granted, go to the monitoring URL (will be given by Corporate IT team) and use the following query in the search box:

```
index="rh_tekton_pipeline" tkn_namespace_name="stonesoup-int-srvc"
```

As of May.2023 no logs for Internal Services are sent to Splunk.

## Testing the operator using Kind

The following steps were tested on Fedora 38 as it runs seamlessly but it may be reproduced in any Linux distribution.

For RHEL users: Extra steps might be required ([Kind - Rootless](https://kind.sigs.k8s.io/docs/user/rootless/)) but as of May 2023 there was no success running Kind after applying them.

### Installing `kind`
```
$ go install sigs.k8s.io/kind@v0.19.0
``` 

### Installing `kubectl`

The following link shows the steps to install kubectl: https://kubernetes.io/docs/tasks/tools/

### Creating the clusters

Create the `local` and `remote` clusters and install tekton latest on them.
Use two different terminals for that - one for each cluster.

#### `local` cluster terminal
```
$ mkdir ~/.kube
$ export KUBECONFIG=~/.kube/local
$ kind create cluster --name=local
$ kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```
#### `remote` cluster terminal
```
$ export KUBECONFIG=~/.kube/remote
$ kind create cluster --name=remote
$ kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```
### Clone and run the operator
```
$ git clone git@github.com:redhat-appstudio/internal-services.git
Cloning into 'internal-services'...
```
#### Installing the internal-services CR

Both clusters should have the CRs installed, so the following steps should be done in both terminals, local and remote.
```
$ cd internal-services
internal-services $ make manifests generate
internal-services $ make install
```

#### Running the operator in the cluster

The operator needs to run a single cluster - in this test case, the `local` cluster/terminal - and it
will listen for `InternalRequests` created in the `remote` cluster denoted by the `--remote-cluster-config-file` parameter. 

It also requires the configuration set in a `InternalServicesConfig` resource - still in the local cluster - with the following content:

```
cat > config.yaml  <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: InternalServicesConfig
metadata:
  name: "config"
spec:
  allowList:
    - default
  debug: true
  volumeClaim:
    name: workspace
    size: 1Gi
EOF
$ kubectl create -f config.yaml
```

When using `kind` the namespace used is the `default` namespace so that is why it is listed in the `allowList` parameter. Should the operator run
in other namespace than the `default` namespace the items in `allowList` should be set accordingly.

Note that the `InternalServiceConfig` *MUST* be named `config`.

Then you can start the operator:


```
internal-services $ go run main.go --remote-cluster-config-file=$HOME/.kube/remote
```

### Testing an InternalRequest

To process a `InternalRequest` the `internal-services-controller` requires a Pipeline created in the Cluster that the request should run in.

#### Sample Pipeline

In this example the following `sample` Pipeline should exist in the `local` cluster.
```
$ cat > pipeline.yaml <<EOF
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: sample
spec:
  tasks:
  - name: sample
    taskSpec:
      results:
      - name: output
        description: just an example
      steps:
      - image: ubuntu
        script:
          echo "Output from today's demo" | tee $(results.output.path)
  results:
  - name: foo
    description: result from the task
    value: $(tasks.sample.results.output)
EOF
$ kubectl create -f pipeline.yaml

```
#### Sample InternalRequest

An `InternalRequest` should be created in the `remote` cluster so the `internal-services-controller` can read and process it from the `local` cluster.
The operator then creates a `PipelineRun` out of the defined `Pipeline` in the `InternalRequest` parameter `request`.

Below follows one sample `InternalRequest`

```
$ cat > internalrequest.yaml <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: InternalRequest
metadata:
  name: "myrequest"
spec:
  request: "sample"
  params:
    foo: bar
    baz: qux
EOF
$ kubectl create -f internalrequest.yaml
```

Once an `InternalRequest` is created and its `PipelineRun` is completed you should see their results in the `InternalRequest` results.

```
$ kubectl get internalrequest myrequest -o yaml
apiVersion: appstudio.redhat.com/v1alpha1
kind: InternalRequest
metadata:
  creationTimestamp: "2023-01-12T15:34:07Z"
  generation: 1
  name: myrequest
  namespace: default
  resourceVersion: "350930396"
  uid: 9c1a56c6-7ee7-4b4d-aebe-bf1a67af0b48
spec:
  params:
    foo: bar
    baz: qux
  request: sample
status:
  completionTime: "2023-01-12T15:34:12Z"
  conditions:
  - lastTransitionTime: "2023-01-12T15:34:12Z"
    message: ""
    reason: Succeeded
    status: "True"
    type: InternalRequestSucceeded
  results:
    foo: |
      Output from today's demo
  startTime: "2023-01-12T15:34:12Z"
```

## Links

- [Configuring the RHTAP Internal Services system](https://docs.google.com/document/d/1ElMVAgHEZ0NHcy9QAJtJAd6LrSY_n8Al18oMPE3STm0/preview)
- [Gain view access to RHTAP logs](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/stonesoup/sop/getting-access.md#gain-view-access-to-rhtap-logs)
- [Splunk & app-sre](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/app-sre/splunk.md#getting-support-working-with-it)
- [Kind - Rootless](https://kind.sigs.k8s.io/docs/user/rootless/)
