# AGENTS.md

Internal Services is a Kubernetes operator that watches InternalRequest CRDs on a remote cluster and creates Tekton PipelineRuns on the local cluster to fulfill them. API group: `appstudio.redhat.com/v1alpha1`. Two CRDs: InternalRequest, InternalServicesConfig.

## Technology Stack

- **Language**: Go
- **Framework**: controller-runtime (Kubebuilder)
- **CRDs**: InternalRequest, InternalServicesConfig
- **Pipeline engine**: Tekton PipelineRuns
- **Testing**: Ginkgo/Gomega + envtest (local K8s API server)
- **Build**: `make test` (unit), `make manifests generate` (codegen)

## Repository Structure

```
api/v1alpha1/          # CRD types, conditions, deepcopy
controllers/           # Reconciler and adapter (single controller: internalrequest)
loader/                # ObjectLoader interface — abstracts K8s resource fetching
tekton/                # PipelineRun builder, status helpers, watch predicates
metadata/              # Label constants (pipelines.appstudio.openshift.io/type)
metrics/               # Prometheus gauge, histogram, counter for request lifecycle
config/                # Kustomize manifests (CRDs, RBAC, manager, samples)
main.go                # Entry point — registers controller, remote cluster, scheme
```

## Architecture

### Reconciliation Pipeline
Operations are chained via `controller.ReconcileHandler()` from operator-toolkit. Each `Ensure*` returns `(controller.OperationResult, error)` — use the toolkit helpers (`ContinueProcessing`, `RequeueWithError`, `RequeueOnErrorOrContinue`, `RequeueOnErrorOrStop`, `StopProcessing`), not raw `ctrl.Result`. New logic goes as an `Ensure*` method on Adapter, wired into the operation slice in `controller.go` — order defines the reconciliation sequence.

PipelineRuns are watched via `EnqueueRequestForAnnotation` — PipelineRun updates trigger re-reconciliation of the owning InternalRequest. PipelineRun lookup is label-based (`MatchingLabels` with name+namespace labels), not owner-reference based.

### Multi-Cluster
`r.Client` (remote cluster) reads/writes InternalRequests. `r.InternalClient` (local cluster) manages PipelineRuns, Pipelines, and InternalServicesConfig. Do not confuse them — status patches go to `r.Client`, PipelineRun CRUD goes to `r.InternalClient`.

### Resource Loading
`loader.ObjectLoader` centralizes all K8s Gets/Lists. A mock implementation (`loader_mock.go`) uses context-key injection via operator-toolkit's `GetMockedResourceAndErrorFromContext` for testing.

## Development Guidelines

- **Git**: conventional commits — `type(scope): description` (72-char title, 88-char body). Enforced by gitlint
- **Linter**: staticcheck (not golangci-lint), no config file
- **API changes**: edit types in `api/v1alpha1/`, then `make generate manifests`
- **Controller changes**: implement in `controllers/internalrequest/adapter.go`
- **Tests**: Ginkgo + envtest alongside code; each package has `*_suite_test.go`
- **New loader method**: update both `loader.go` (interface + impl) and `loader_mock.go`

## Key Patterns

- **Adapter pattern:** the controller delegates to an **adapter** (`adapter.go`) that holds two K8s clients, the resource under reconciliation, an `ObjectLoader`, and a logger. All domain logic lives in adapter methods, not in the controller.
- **Idempotent operations**: every `Ensure*` function starts with gate conditions that skip work already done (e.g. `HasCompleted()`) or bail early when prerequisites aren't met, making reconciliation safe to re-run at any point
- **Patching discipline**: status is updated via `client.Status().Patch()` with `client.MergeFrom(obj.DeepCopy())` — only in state-changing operations
- **Debug mode**: when `InternalServicesConfig.Spec.Debug` is true, PipelineRun cleanup is skipped for inspection
- **PipelineRunBuilder** (`tekton/pipeline_run.go`): fluent API — `NewInternalRequestPipelineRun().WithOwner().WithPipelineRef().WithInternalRequest().AsPipelineRun()`
- **Cache filtering**: PipelineRuns in the manager cache are filtered by label `pipelines.appstudio.openshift.io/type: release`
- **Config singleton**: InternalServicesConfig must be named `"config"` (`InternalServicesConfigResourceName`), loaded from `SERVICE_NAMESPACE` env var (defaults to `"default"`)
