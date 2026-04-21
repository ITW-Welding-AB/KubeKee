package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ITW-Welding-AB/KubeKee/api/v1alpha1"
	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KeePassSourceReconciler reconciles a KeePassSource object.
type KeePassSourceReconciler struct {
	client.Client
}

func (r *KeePassSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var source v1alpha1.KeePassSource
	if err := r.Get(ctx, req.NamespacedName, &source); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if source.Spec.Suspend {
		logger.Info("reconciliation suspended")
		return ctrl.Result{}, nil
	}

	// Get password from secret
	password, err := r.getPassword(ctx, source.Namespace, source.Spec.PasswordSecretRef)
	if err != nil {
		r.setCondition(&source, "Ready", metav1.ConditionFalse, "PasswordError", err.Error())
		_ = r.Status().Update(ctx, &source)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Resolve the .kdbx file path
	dbPath, revision, err := r.resolveDBPath(ctx, &source)
	if err != nil {
		r.setCondition(&source, "Ready", metav1.ConditionFalse, "SourceError", err.Error())
		_ = r.Status().Update(ctx, &source)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Skip if the revision hasn't changed (source-based reconciliation)
	if revision != "" && revision == source.Status.SourceArtifactRevision {
		logger.Info("source artifact revision unchanged, skipping", "revision", revision)
		return ctrl.Result{RequeueAfter: r.getInterval(source.Spec.Interval)}, nil
	}

	// Open KeePass DB
	db, err := kdbx.OpenDB(dbPath, password)
	if err != nil {
		r.setCondition(&source, "Ready", metav1.ConditionFalse, "DBOpenError", err.Error())
		_ = r.Status().Update(ctx, &source)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Get entries with filtering
	entries := r.filterEntries(db, source.Spec.Groups, source.Spec.Entries)

	// Apply entries
	applied := 0
	for _, entry := range entries {
		if entry.Content == "" {
			continue
		}
		if err := r.applyEntry(ctx, entry, source.Spec.TargetNamespace); err != nil {
			logger.Error(err, "failed to apply entry", "title", entry.Title, "group", entry.Group)
			continue
		}
		applied++
	}

	now := metav1.Now()
	source.Status.LastSyncTime = &now
	source.Status.AppliedEntries = applied
	if revision != "" {
		source.Status.SourceArtifactRevision = revision
	}
	r.setCondition(&source, "Ready", metav1.ConditionTrue, "Synced", fmt.Sprintf("Applied %d entries", applied))
	if err := r.Status().Update(ctx, &source); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconciliation complete", "applied", applied, "revision", revision)
	return ctrl.Result{RequeueAfter: r.getInterval(source.Spec.Interval)}, nil
}

// resolveDBPath determines the .kdbx file path, either from spec.dbPath
// or by resolving a Flux GitRepository / ArgoCD Application source.
func (r *KeePassSourceReconciler) resolveDBPath(ctx context.Context, source *v1alpha1.KeePassSource) (string, string, error) {
	if source.Spec.SourceRef == nil {
		if source.Spec.DBPath == "" {
			return "", "", fmt.Errorf("either dbPath or sourceRef must be specified")
		}
		return source.Spec.DBPath, "", nil
	}

	ref := source.Spec.SourceRef
	ns := ref.Namespace
	if ns == "" {
		ns = source.Namespace
	}

	switch ref.Kind {
	case "GitRepository":
		return r.resolveFluxGitRepository(ctx, ref.Name, ns, source.Spec.DBFileName)
	case "Application":
		return r.resolveArgoCDApplication(ctx, ref.Name, ns, source.Spec.DBFileName)
	default:
		return "", "", fmt.Errorf("unsupported sourceRef kind %q, must be GitRepository or Application", ref.Kind)
	}
}

// resolveFluxGitRepository reads a Flux GitRepository to get the artifact
// URL and revision, then downloads the .kdbx file from the artifact.
func (r *KeePassSourceReconciler) resolveFluxGitRepository(ctx context.Context, name, namespace, dbFileName string) (string, string, error) {
	logger := log.FromContext(ctx)

	gitRepo := &unstructured.Unstructured{}
	gitRepo.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "GitRepository",
	})
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, gitRepo); err != nil {
		return "", "", fmt.Errorf("getting Flux GitRepository %s/%s: %w", namespace, name, err)
	}

	// Extract artifact info from status
	status, ok := gitRepo.Object["status"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("GitRepository %s/%s has no status", namespace, name)
	}
	artifact, ok := status["artifact"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("GitRepository %s/%s has no artifact", namespace, name)
	}

	revision, _ := artifact["revision"].(string)
	artifactURL, _ := artifact["url"].(string)
	if artifactURL == "" {
		return "", "", fmt.Errorf("GitRepository %s/%s artifact has no URL", namespace, name)
	}

	logger.Info("resolved Flux GitRepository", "name", name, "revision", revision)

	dbPath, err := r.downloadArtifact(artifactURL, dbFileName)
	if err != nil {
		return "", "", fmt.Errorf("downloading Flux artifact: %w", err)
	}
	return dbPath, revision, nil
}

// resolveArgoCDApplication reads an ArgoCD Application to find the sync
// revision and locate the .kdbx file on a shared volume.
func (r *KeePassSourceReconciler) resolveArgoCDApplication(ctx context.Context, name, namespace, dbFileName string) (string, string, error) {
	logger := log.FromContext(ctx)

	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, app); err != nil {
		return "", "", fmt.Errorf("getting ArgoCD Application %s/%s: %w", namespace, name, err)
	}

	status, _ := app.Object["status"].(map[string]interface{})
	sync, _ := status["sync"].(map[string]interface{})
	revision, _ := sync["revision"].(string)

	spec, _ := app.Object["spec"].(map[string]interface{})
	appSource, _ := spec["source"].(map[string]interface{})
	repoPath, _ := appSource["path"].(string)

	logger.Info("resolved ArgoCD Application", "name", name, "revision", revision, "path", repoPath)

	if dbFileName == "" {
		dbFileName = "secrets.kdbx"
	}

	candidates := []string{
		filepath.Join("/data", repoPath, dbFileName),
		filepath.Join("/data", dbFileName),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, revision, nil
		}
	}

	return "", "", fmt.Errorf("could not find %s in ArgoCD paths: %v", dbFileName, candidates)
}

// downloadArtifact downloads a file from a Flux source-controller artifact URL.
func (r *KeePassSourceReconciler) downloadArtifact(artifactURL, dbFileName string) (string, error) {
	if dbFileName == "" {
		dbFileName = "secrets.kdbx"
	}

	resp, err := http.Get(artifactURL)
	if err != nil {
		return "", fmt.Errorf("fetching artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("artifact returned status %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "kubekee-artifact-*")
	if err != nil {
		return "", err
	}

	outPath := filepath.Join(tmpDir, dbFileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return "", fmt.Errorf("writing artifact: %w", err)
	}

	return outPath, nil
}

// filterEntries filters DB entries based on group and title filters.
func (r *KeePassSourceReconciler) filterEntries(db *kdbx.DB, groups, entries []string) []kdbx.Entry {
	if len(groups) == 0 && len(entries) == 0 {
		return db.ListEntries("")
	}

	var result []kdbx.Entry
	if len(groups) > 0 {
		for _, group := range groups {
			result = append(result, db.ListEntries(group)...)
		}
	} else {
		result = db.ListEntries("")
	}

	if len(entries) > 0 {
		titleSet := make(map[string]bool, len(entries))
		for _, e := range entries {
			titleSet[e] = true
		}
		var filtered []kdbx.Entry
		for _, e := range result {
			if titleSet[e.Title] {
				filtered = append(filtered, e)
			}
		}
		return filtered
	}

	return result
}

func (r *KeePassSourceReconciler) getPassword(ctx context.Context, namespace string, ref v1alpha1.SecretKeyRef) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, &secret); err != nil {
		return "", fmt.Errorf("getting password secret %q: %w", ref.Name, err)
	}
	data, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %q", ref.Key, ref.Name)
	}
	return string(data), nil
}

func (r *KeePassSourceReconciler) applyEntry(ctx context.Context, entry kdbx.Entry, targetNS string) error {
	obj := &unstructured.Unstructured{}

	// Try JSON first, then YAML
	if err := json.Unmarshal([]byte(entry.Content), &obj.Object); err != nil {
		decoder := utilyaml.NewYAMLOrJSONDecoder(
			strings.NewReader(entry.Content), 4096,
		)
		if err := decoder.Decode(&obj.Object); err != nil {
			return fmt.Errorf("parsing entry %q: %w", entry.Title, err)
		}
	}

	// Override namespace if specified
	if targetNS != "" {
		obj.SetNamespace(targetNS)
	}

	// Server-side apply
	existing := obj.DeepCopy()
	err := r.Get(ctx, types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}, existing)

	if apierrors.IsNotFound(err) {
		return r.Create(ctx, obj)
	} else if err != nil {
		return err
	}

	// Update existing
	obj.SetResourceVersion(existing.GetResourceVersion())
	return r.Update(ctx, obj)

}

func (r *KeePassSourceReconciler) setCondition(source *v1alpha1.KeePassSource, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&source.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (r *KeePassSourceReconciler) getInterval(interval string) time.Duration {
	if interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			return d
		}
	}
	return 5 * time.Minute
}

func (r *KeePassSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KeePassSource{}).
		Complete(r)
}
