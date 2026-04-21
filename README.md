<div>
    <img align="right" width="80" alt="Elga logo RGB" src="https://github.com/ITW-Welding-AB/.github/blob/main/images/Elga-logo-RGB.png?raw=true" />
</div>
<br clear="left"/>
<div align="center">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://github.com/ITW-Welding-AB/.github/blob/main/images/ITW-welding-white.png?raw=true">
      <source media="(prefers-color-scheme: light)" srcset="https://github.com/ITW-Welding-AB/.github/blob/main/images/ITW-welding.png?raw=true">
      <img alt="ITW Welding" width="800" src="https://github.com/ITW-Welding-AB/.github/blob/main/images/ITW-welding-white.png?raw=true">
    </picture>
</div>

<br clear="left"/>

<div align="center">

# ITW Welding AB

[Website](https://elgawelding.com/) | [ITW Welding](https://www.itwwelding.com/) | [Contact Us](https://elgawelding.com/contact-us/)

---

</div>

# KubeKee

K8s KeePass CLI & Operator tool for CI/CD workflows.

Store Kubernetes manifests (Secrets, ConfigMaps, etc.) securely inside KeePass (`.kdbx`) databases. Use the CLI to
import, export, edit, and list entries. Deploy the operator to automatically sync entries to your cluster alongside Flux
or ArgoCD.

## Getting Started

### Install via Go

```bash
go install github.com/ITW-Welding-AB/KubeKee/cmd/kubekee@latest
```

### Build from source

```bash
git clone https://github.com/ITW-Welding-AB/KubeKee.git
cd KubeKee
go mod tidy
go build -o kubekee ./cmd/kubekee
```

## CLI Usage

### Initialize a new KeePass database

```bash
kubekee init --db secrets.kdbx --password mypassword
# or use env var
export KUBEKEE_PASSWORD=mypassword
kubekee init --db secrets.kdbx
```

### Import YAML/JSON files

```bash
kubekee import secret.yaml --db secrets.kdbx --password mypassword
kubekee import deployment.yaml service.yaml --db secrets.kdbx --group production
```

### List entries

```bash
kubekee list --db secrets.kdbx --password mypassword
kubekee list --db secrets.kdbx --group production
```

### Export an entry

```bash
kubekee export my-secret --db secrets.kdbx --password mypassword          # stdout
kubekee export my-secret --db secrets.kdbx -o my-secret.yaml              # to file
```

### Edit an entry

```bash
kubekee edit my-secret --db secrets.kdbx --password mypassword
# Opens $EDITOR, saves changes back to the database
```

---

## Helm Installation

### Install with Helm

```bash
helm install kubekee charts/kubekee -n kubekee-system --create-namespace
```

### Integration Modes

KubeKee supports four source modes via `sourceMode`:

| Mode      | Description                                       |
|-----------|---------------------------------------------------|
| `none`    | Direct file path — mount the `.kdbx` via a volume |
| `gitSync` | Built-in git-sync sidecar pulls the repo          |
| `flux`    | Integrates with an existing Flux `GitRepository`  |
| `argocd`  | Integrates with an existing ArgoCD `Application`  |

### Flux Integration

If you already have Flux installed with a `GitRepository` that syncs your infrastructure repo containing a `.kdbx` file:

```bash
helm install kubekee charts/kubekee -n kubekee-system --create-namespace \
  -f charts/kubekee/examples/flux-values.yaml \
  --set flux.gitRepository.name=my-infrastructure \
  --set keepassSource.passwordSecretRef.name=kubekee-password
```

This creates a `KeePassSource` CR that references your Flux `GitRepository`. The operator reads the artifact URL from
the GitRepository status, downloads the `.kdbx` file, and applies entries to the cluster. It tracks the artifact
revision and only re-syncs when the source changes.

```yaml
apiVersion: kubekee.itwwelding.com/v1alpha1
kind: KeePassSource
metadata:
  name: my-secrets
spec:
  sourceRef:
    kind: GitRepository
    name: my-infrastructure
    namespace: flux-system
  dbFileName: secrets.kdbx
  passwordSecretRef:
    name: kubekee-password
    key: password
  targetNamespace: production
  interval: 5m
```

### ArgoCD Integration

If you have ArgoCD syncing your repo:

```bash
helm install kubekee charts/kubekee -n kubekee-system --create-namespace \
  -f charts/kubekee/examples/argocd-values.yaml \
  --set argocd.application.name=my-infrastructure \
  --set gitSync.repo=https://github.com/your-org/your-repo.git
```

The operator reads the ArgoCD Application to track the sync revision, while a git-sync sidecar provides the actual
`.kdbx` file on a shared volume.

```yaml
apiVersion: kubekee.itwwelding.com/v1alpha1
kind: KeePassSource
metadata:
  name: my-secrets
spec:
  sourceRef:
    kind: Application
    name: my-infrastructure
    namespace: argocd
  dbFileName: secrets.kdbx
  passwordSecretRef:
    name: kubekee-password
    key: password
  interval: 5m
```

### Standalone git-sync

No Flux or ArgoCD? Use the built-in git-sync sidecar:

```bash
helm install kubekee charts/kubekee -n kubekee-system --create-namespace \
  -f charts/kubekee/examples/gitsync-values.yaml \
  --set gitSync.repo=https://github.com/your-org/your-repo.git
```

### Entry Filtering

You can filter which KeePass entries get applied:

```yaml
keepassSource:
  create: true
  entries: # Only sync specific entries by title
    - my-secret
    - my-configmap
  groups: # Only sync entries from specific groups
    - production
```

### Password Secret

Create the password secret before installing:

```bash
kubectl create secret generic kubekee-password \
  -n kubekee-system \
  --from-literal=password=your-keepass-password
```

---

## Operator (without Helm)

### Install CRD

```bash
kubectl apply -f config/crd/keepasssource.yaml
```

### Deploy

```bash
kubectl apply -f config/manager/deployment.yaml
```

### Run operator locally

```bash
kubekee operator
```

## Docker

```bash
docker build -t kubekee .
docker run kubekee init --db /data/secrets.kdbx --password mypassword
```

## License

See [LICENSE](LICENSE).
