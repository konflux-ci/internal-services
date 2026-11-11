# E2E infrastructure

`simple-e2e-pipeline.yaml` provisions separate local and remote Kind clusters
on IBM Cloud using catalog Task `kind-ibm-provision` 0.4, then destroys them
with `kind-ibm-deprovision` 0.1. Both Tasks and their nested StepActions use
catalog commit `2d0243d6b0ad1e6f48a2eadd7c5cc41c39b8b52f` by default.

Each cluster uses Kubernetes `v1.37` and the IBM Flex profile `cxf-32x64`.
Control-plane hardening is enabled with the catalog's tuning for that profile.
A short ID derived from the PipelineRun UID gets a `-local` or `-remote`
suffix, keeping the clusters separate and within IBM's resource-name limit.

## Credentials and state

The CI namespace must contain the Secret selected by
`ibmcloud-credentials-secret` (default `mapt-kind-secret-ibm`), with these keys:

- `IBMCLOUD_API_KEY`
- `IBMCLOUD_COS_ACCESS_KEY_ID`
- `IBMCLOUD_COS_SECRET_ACCESS_KEY`
- Optional: `IBMCLOUD_COS_ENDPOINT`

`ibmcloud-backend-url` defaults to `s3://konflux-mapt-state/mapt/kind`.
The COS bucket must already exist and be accessible to those credentials.
The pipeline appends each cluster's ID, including its role, to this prefix
and passes the same state URL to its provision and deprovision Tasks.

The existing `konflux-test-infra-secret` parameter selects the registry
credentials used for the provision and cleanup artifacts.

## Diagnostics

The existing OCI artifact tags remain `<PipelineRun name>-local` and
`<PipelineRun name>-remote`. Each provision Task uploads
`kind-ibm-provision-<TaskRun UID>.log` and successfully captured helper logs
(`cp-patch-*.log`, `setup-node-*.log`), after secret scanning. Failed runs
also attempt cluster-resource collection during deprovision.

The PR analyzer currently requires an exact provisioning-log filename and
cannot select the IBM Task's UID-based filename. Automatic provisioning-log
analysis needs a separate analyzer update; the logs remain available in the
OCI artifacts unless secret scanning removes them.
