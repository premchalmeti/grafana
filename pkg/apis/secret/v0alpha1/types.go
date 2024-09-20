package v0alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// When writing values, only one property is valid at a time
// When reading, GUID will always be set, the Value+Ref *may* be set
type SecureValue struct {
	// GUID is a unique identifier for this exact field
	// it must match the same group+resource+namespace+name where it was created
	GUID string `json:"guid,omitempty"`

	// The raw non-encrypted value
	// Used when writing new values, or reading decrypted values
	Value string `json:"value,omitempty"`

	// // Used when linking this value to a known (and authorized) reference id
	// // Enterprise only????
	// Ref string `json:"ref,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecureValues struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SecureValuesSpec `json:"spec,omitempty"`
}

type SecureValuesSpec struct {
	Title       string `json:"title"`
	Description string `json:"description"`

	// Values
	// These are not returned in k8s get/list responses
	Values map[string]SecureValue `json:"values"`

	// List of groups authorized to decrypt these values
	// support wildcards?
	// will be compared to the access token when trying to decrypt
	AuthorizedApps []string `json:"authorized"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecureValuesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecureValues `json:"items,omitempty"`
}
