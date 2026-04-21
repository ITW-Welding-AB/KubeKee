package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// KeePassSourceSpec defines the desired state of KeePassSource.
type KeePassSourceSpec struct {
	// DBPath is the path to the .kdbx file (mounted volume).
	// Used when no sourceRef is provided.
	DBPath string `json:"dbPath,omitempty"`

	// SourceRef references a Flux GitRepository or ArgoCD Application
	// whose artifact contains the .kdbx file.
	SourceRef *SourceRef `json:"sourceRef,omitempty"`

	// DBFileName is the name of the .kdbx file within the source artifact.
	// Used together with sourceRef. Defaults to "secrets.kdbx".
	DBFileName string `json:"dbFileName,omitempty"`

	// PasswordSecretRef references a Secret containing the DB password.
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`

	// TargetNamespace is the namespace to apply resources to.
	// If empty, uses the resource's own namespace metadata.
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Interval is the reconciliation interval (e.g. "5m", "1h").
	Interval string `json:"interval,omitempty"`

	// Suspend stops reconciliation when true.
	Suspend bool `json:"suspend,omitempty"`

	// Entries filter: only sync entries matching these titles (empty = all).
	Entries []string `json:"entries,omitempty"`

	// Groups filter: only sync entries in these groups (empty = all).
	Groups []string `json:"groups,omitempty"`
}

// SourceRef references a Flux GitRepository or ArgoCD Application.
type SourceRef struct {
	// Kind of the source: "GitRepository" (Flux) or "Application" (ArgoCD).
	Kind string `json:"kind"`
	// Name of the source object.
	Name string `json:"name"`
	// Namespace of the source object.
	Namespace string `json:"namespace,omitempty"`
}

// SecretKeyRef references a key in a Secret.
type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// KeePassSourceStatus defines the observed state of KeePassSource.
type KeePassSourceStatus struct {
	// LastSyncTime is the last time the source was synced.
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represents the current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// AppliedEntries is the count of entries applied.
	AppliedEntries int `json:"appliedEntries,omitempty"`

	// SourceArtifactRevision is the last observed revision from the source.
	SourceArtifactRevision string `json:"sourceArtifactRevision,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DB Path",type=string,JSONPath=`.spec.dbPath`
// +kubebuilder:printcolumn:name="Interval",type=string,JSONPath=`.spec.interval`
// +kubebuilder:printcolumn:name="Last Sync",type=date,JSONPath=`.status.lastSyncTime`

// KeePassSource is the Schema for the keepasssources API.
type KeePassSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeePassSourceSpec   `json:"spec,omitempty"`
	Status KeePassSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeePassSourceList contains a list of KeePassSource.
type KeePassSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeePassSource `json:"items"`
}

// DeepCopyObject implementations for runtime.Object interface.
func (in *KeePassSource) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

func (in *KeePassSource) DeepCopy() *KeePassSource {
	if in == nil {
		return nil
	}
	out := new(KeePassSource)
	in.DeepCopyInto(out)
	return out
}

func (in *KeePassSource) DeepCopyInto(out *KeePassSource) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *KeePassSourceSpec) DeepCopyInto(out *KeePassSourceSpec) {
	*out = *in
	if in.SourceRef != nil {
		sr := *in.SourceRef
		out.SourceRef = &sr
	}
	if in.Entries != nil {
		out.Entries = make([]string, len(in.Entries))
		copy(out.Entries, in.Entries)
	}
	if in.Groups != nil {
		out.Groups = make([]string, len(in.Groups))
		copy(out.Groups, in.Groups)
	}
}

func (in *KeePassSourceStatus) DeepCopyInto(out *KeePassSourceStatus) {
	*out = *in
	if in.LastSyncTime != nil {
		t := *in.LastSyncTime
		out.LastSyncTime = &t
	}
	if in.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(in.Conditions))
		for i := range in.Conditions {
			in.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
}

func (in *KeePassSourceList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

func (in *KeePassSourceList) DeepCopy() *KeePassSourceList {
	if in == nil {
		return nil
	}
	out := new(KeePassSourceList)
	in.DeepCopyInto(out)
	return out
}

func (in *KeePassSourceList) DeepCopyInto(out *KeePassSourceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]KeePassSource, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}
