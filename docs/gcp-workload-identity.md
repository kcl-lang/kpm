# GCP Workload Identity for `kpm` OCI pulls

This guide shows how to authenticate `kpm` to Google Container Registry /
Artifact Registry from a GKE pod that uses Workload Identity, without
distributing any static credential.

## Prerequisites

- A GKE cluster with **Workload Identity** enabled (default on GKE 1.16+).
- A GCP service account (`gsa@example.iam.gserviceaccount.com`) that has
  permission to read from your Artifact Registry / GCR repository.
  The minimum role is `roles/artifactregistry.reader`.
- A Kubernetes service account (`ksa:default:my-ksa`) in the namespace
  where `kpm` runs.

## One-time setup

1. **Create the GCP service account and grant access** to the
   repository (skip this if you already have one):

   ```bash
   gcloud iam service-accounts create kpm-puller \
       --display-name="kpm puller"

   PROJECT=$(gcloud config get-value project)

   gcloud projects add-iam-policy-binding "$PROJECT" \
       --member="serviceAccount:kpm-puller@$PROJECT.iam.gserviceaccount.com" \
       --role="roles/artifactregistry.reader"
   ```

2. **Bind the Kubernetes SA to the GCP SA**:

   ```bash
   gcloud iam service-accounts add-iam-policy-binding \
       "kpm-puller@$PROJECT.iam.gserviceaccount.com" \
       --role="roles/iam.workloadIdentityUser" \
       --member="serviceAccount:$PROJECT.svc.id.goog[my-namespace/my-ksa]"
   ```

   And annotate the KSA so GKE injects the right service-account token:

   ```bash
   kubectl annotate serviceaccount my-ksa \
       -n my-namespace \
       iam.gke.io/gcp-service-account=kpm-puller@$PROJECT.iam.gserviceaccount.com
   ```

3. **Run `kpm` from a pod that uses that KSA**. No `imagePullSecrets`
   are required; the metadata server returns a federated identity that
   GCR / Artifact Registry accepts.

## Logging in

From inside the pod (or anywhere on GCE/GKE with Workload Identity):

```bash
kpm login gcr.io/<project>/<repo> --provider=gcp
```

What this does:

1. Reads an identity token from the GCE metadata server
   (`169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token`).
2. Exchanges it for an OAuth2 access token scoped to
   `https://gcr.io/<project>/<repo>`.
3. Writes the credential under the `gcr.io/<project>/<repo>` key in
   `$KCL_PKG_PATH/.kpm/config/config.json` via the same ORAS store
   `kpm login` already uses.

Subsequent `kpm pull` and `kpm push` commands pick the credential up
the same way they do for username/password logins.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `kpm auth: not running on GCE/GKE` | Pod is not on GCE, or Workload Identity binding is missing. |
| `failed to login '…', please check registry, username and password is valid` | The GCP SA does not have `roles/artifactregistry.reader` on the project, or the audience was wrong. |
| `kpm auth: empty GCP access token` | The metadata server returned a payload without an `access_token` field — usually a transient IAM failure. |

## Token lifetime

GCP access tokens minted from the metadata server have a ~1 hour TTL.
`kpm login` writes whatever token is current at login time; for long-
running workloads you should refresh via the same `--provider=gcp`
flag rather than reusing a stale `~/.kcl/pkg/.kpm/config/config.json`
entry.

## Custom audiences

If you need a custom audience (rare — GCR / Artifact Registry accept
the default `https://<registry>`), set it via the `--audience` flag
on a follow-up PR. Today only the default audience is supported.