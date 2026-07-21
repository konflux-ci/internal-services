---
name: create-operation
description: Conventions for adding a new operation to the InternalRequest controller's reconciliation pipeline.
---

# Creating a New Operation

An **operation** is a method on the Adapter struct that represents one step in the controller's reconciliation pipeline. Operations are executed sequentially by `controller.ReconcileHandler()` from operator-toolkit. Each operation must be idempotent — safe to re-run at any point if the reconciliation is interrupted and restarted.

## Operation signature

Every operation must have this exact signature:

```go
func (a *Adapter) EnsureSomethingHappens() (controller.OperationResult, error)
```

The name must start with `Ensure` and describe the desired end state, not the action taken. Examples: `EnsureFinalizerIsAdded`, `EnsureRequestIsAllowed`, `EnsurePipelineRunIsCreated`.

## Return values

Use the result helpers from `github.com/konflux-ci/operator-toolkit/controller`:

| Helper | When to use |
|---|---|
| `controller.ContinueProcessing()` | Work done or skipped — proceed to next operation |
| `controller.StopProcessing()` | Reconciliation should end (e.g. resource already completed) |
| `controller.Requeue()` | Requeue immediately, stop remaining operations |
| `controller.RequeueWithError(err)` | Requeue due to error |
| `controller.RequeueOnErrorOrContinue(err)` | If err is non-nil, requeue; otherwise continue. **Most common for status patches** |
| `controller.RequeueOnErrorOrStop(err)` | If err is non-nil, requeue; otherwise stop |
| `controller.RequeueAfter(delay, err)` | Requeue after a specific delay |

## Operation structure

Every operation follows this skeleton:

```go
// EnsureXxxIsYyy is an operation that will ensure that <describe the desired state>.
func (a *Adapter) EnsureXxxIsYyy() (controller.OperationResult, error) {
    // 1. Gate condition — skip if already done or prerequisites not met
    if a.internalRequest.HasCompleted() {
        return controller.ContinueProcessing()
    }

    // 2. Load resources via the loader
    resource, err := a.loader.GetSomeResource(a.ctx, a.internalClient, ...)
    if err != nil && !errors.IsNotFound(err) {
        return controller.RequeueWithError(err)
    }

    // 3. Perform the work (create objects, update state, etc.)

    // 4. Patch status and return
    patch := client.MergeFrom(a.internalRequest.DeepCopy())
    a.internalRequest.MarkRunning()
    return controller.RequeueOnErrorOrContinue(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
}
```

## Gate conditions and idempotency

Every operation must start with a gate condition that checks whether the work has already been done. This makes reconciliation safe to restart at any point.

Common gate patterns:

```go
// Already completed — skip or stop
if a.internalRequest.HasCompleted() {
    return controller.ContinueProcessing()
}

// Already completed — stop entire pipeline
if a.internalRequest.HasCompleted() {
    return controller.StopProcessing()
}

// Resource already exists and request is already running — skip
pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
if pipelineRun != nil && a.internalRequest.IsRunning() {
    return controller.ContinueProcessing()
}

// Deletion not in progress — skip
if a.internalRequest.GetDeletionTimestamp() == nil {
    return controller.ContinueProcessing()
}
```

## Multi-cluster client discipline

This operator uses two Kubernetes clients. Using the wrong client is a common mistake:

| Client | Cluster | Used for |
|---|---|---|
| `a.client` | Remote | InternalRequest reads, status patches, metadata patches (finalizers) |
| `a.internalClient` | Local | PipelineRun CRUD, Pipeline reads, InternalServicesConfig reads |

Rules:
- Status patches on InternalRequest → `a.client.Status().Patch()`
- Metadata patches on InternalRequest (finalizers) → `a.client.Patch()`
- PipelineRun create/delete/patch → `a.internalClient`
- All loader calls for PipelineRuns, Pipelines, and Config pass `a.internalClient`

## Status patching discipline

Always use the merge-from-deep-copy pattern:

```go
patch := client.MergeFrom(a.internalRequest.DeepCopy())
// ... mutate a.internalRequest.Status fields ...
return controller.RequeueOnErrorOrContinue(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
```

Rules:
- Use `a.client.Status().Patch()` for status subresource updates on InternalRequest.
- Use `a.client.Patch()` for metadata updates (finalizers, labels, annotations) on InternalRequest.
- Use `a.internalClient.Patch()` for PipelineRun spec patches (e.g. cancellation).
- Only patch in operations that change state. Read-only and tracking operations that find no changes must NOT patch.
- Take the `DeepCopy()` **before** any mutations — this is the baseline for the merge patch.

## Error handling

Errors are handled inline with the toolkit return helpers. There is no centralized error handler in this project — each operation classifies errors directly:

```go
// Loader errors — distinguish NotFound from real errors
resource, err := a.loader.GetSomeResource(a.ctx, a.internalClient, ...)
if err != nil && !errors.IsNotFound(err) {
    return controller.RequeueWithError(err)
}

// Creation errors — requeue with error
pipelineRun, err := a.createInternalRequestPipelineRun()
if err != nil {
    return controller.RequeueWithError(err)
}

// Status patch errors — requeue on error, otherwise continue
return controller.RequeueOnErrorOrContinue(a.client.Status().Patch(a.ctx, a.internalRequest, patch))

// Rejection — requeue on error, otherwise stop
return controller.RequeueOnErrorOrStop(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
```

## Loading resources

Always load resources through `a.loader` (the `loader.ObjectLoader` interface), never directly via `a.client.Get()` or `a.internalClient.Get()`. The loader provides:
- A mock implementation for tests via context-key injection

If a new resource type needs to be loaded, update both `loader/loader.go` (interface + real implementation) and `loader/loader_mock.go` (mock implementation with a new context key).

## Creating PipelineRuns

Use the `InternalRequestPipelineRun` builder from the `tekton` package:

```go
pipelineRun := tekton.NewInternalRequestPipelineRun(a.internalServicesConfig).
    WithInternalRequest(a.internalRequest).
    WithOwner(a.internalRequest).
    WithPipelineRef(a.internalRequest, a.internalServicesConfig).
    AsPipelineRun()

err := a.internalClient.Create(a.ctx, pipelineRun)
```

The builder is a fluent API — chain methods and call `.AsPipelineRun()` to get the `*tektonv1.PipelineRun` for use with the K8s client.

## Registering the operation

After implementing the operation, add it to the operation slice in the controller's `Reconcile` method (`controllers/internalrequest/controller.go`):

```go
return controller.ReconcileHandler([]controller.Operation{
    adapter.EnsureFinalizersAreCalled,
    adapter.EnsureFinalizerIsAdded,
    adapter.EnsureRequestINotCompleted,
    adapter.EnsureConfigIsLoaded,
    adapter.EnsureRequestIsAllowed,
    adapter.EnsurePipelineRunIsCreated,
    // ... new operation here ...
    adapter.EnsureStatusIsTracked,
    adapter.EnsurePipelineRunIsDeleted,
})
```

**Order matters.** Operations execute sequentially and any one can short-circuit the pipeline. Place the new operation:
- After its prerequisites (operations whose results it depends on)
- Before operations that depend on its results
- `EnsureConfigIsLoaded` must come before any operation that uses `a.internalServicesConfig`
- `EnsureStatusIsTracked` and `EnsurePipelineRunIsDeleted` are typically last

## Writing tests

Tests use Ginkgo/Gomega with envtest. Each test file lives alongside the code it tests in the same package.

### Test structure

```go
var _ = Describe("Adapter", Ordered, func() {
    var (
        createResources func()
        deleteResources func()
        adapter         *Adapter
    )

    Context("When calling EnsureXxxIsYyy", func() {
        AfterEach(func() { deleteResources() })
        BeforeEach(func() { createResources() })

        It("should continue processing when already completed", func() {
            adapter.internalRequest.MarkSucceeded()
            result, err := adapter.EnsureXxxIsYyy()
            Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
            Expect(err).To(BeNil())
        })
    })
})
```

### Result checking convention

```go
// ContinueProcessing: not cancelled, not requeued
Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())

// StopProcessing: cancelled, not requeued
Expect(result.CancelRequest && !result.RequeueRequest).To(BeTrue())

// Requeue: not cancelled, requeued
Expect(!result.CancelRequest && result.RequeueRequest).To(BeTrue())
```

### Resource lifecycle

Tests use `createResources`/`deleteResources` closures managed by `BeforeEach`/`AfterEach`:

```go
createResources = func() {
    internalRequest = &v1alpha1.InternalRequest{...}
    Expect(k8sClient.Create(ctx, internalRequest)).To(Succeed())

    internalServicesConfig = &v1alpha1.InternalServicesConfig{...}
    Expect(k8sClient.Create(ctx, internalServicesConfig)).To(Succeed())

    adapter = NewAdapter(ctx, k8sClient, k8sClient, internalRequest, loader.NewMockLoader(), logger)
    adapter.internalServicesConfig = internalServicesConfig
}

deleteResources = func() {
    Expect(k8sClient.Delete(ctx, internalServicesConfig)).To(Succeed())
    // Remove finalizers before deleting InternalRequest
    controllerutil.RemoveFinalizer(internalRequest, tekton.InternalRequestFinalizer)
    Expect(k8sClient.Update(ctx, internalRequest)).To(Succeed())
    Expect(k8sClient.Delete(ctx, internalRequest)).To(Succeed())
    Expect(k8sClient.DeleteAllOf(ctx, &tektonv1.PipelineRun{}, ...)).To(Succeed())
}
```

### Mock injection for error testing

Use operator-toolkit's context-based mocking to override specific loader calls:

```go
adapter.ctx = toolkit.GetMockedContext(ctx, []toolkit.MockData{
    {
        ContextKey: loader.InternalRequestPipelineRunContextKey,
        Err:        fmt.Errorf("not found"),
    },
})
```

Available context keys (defined in `loader/loader_mock.go`):
- `loader.InternalRequestContextKey`
- `loader.InternalRequestPipelineContextKey`
- `loader.InternalRequestPipelineRunContextKey`
- `loader.InternalServicesConfigContextKey`

### What to test

For each operation, test:
1. **Gate condition**: returns `ContinueProcessing` when the work is already done or prerequisites aren't met.
2. **Loader errors**: requeues with error when resource loading fails.
3. **Happy path**: performs the work and returns the expected result.
4. **Status patch**: the correct status fields are set after the operation completes.

## Verification

After implementing the operation and its tests:

```bash
go vet ./controllers/internalrequest/
go build ./controllers/internalrequest/
go test ./controllers/internalrequest/
```

Use `go vet` and `go build` for fast iteration on a single package, then run `go test` for the full test suite of that package.

## Checklist

- [ ] Operation method on Adapter with `Ensure` prefix and correct signature
- [ ] Doc comment describing the desired end state
- [ ] Gate condition at the top (idempotency)
- [ ] Correct client used (`a.client` for InternalRequest, `a.internalClient` for PipelineRun/Config)
- [ ] Resources loaded via `a.loader`, not `a.client.Get()` or `a.internalClient.Get()`
- [ ] Status patches use `client.MergeFrom(a.internalRequest.DeepCopy())`
- [ ] Operation registered in the controller's `Reconcile` method in the correct position
- [ ] Tests cover gate conditions, error paths, and happy path
- [ ] `go vet`, `go build`, and `go test` pass for the package
