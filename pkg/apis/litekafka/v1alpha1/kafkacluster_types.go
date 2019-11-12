package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Port defines the desired state of Port
// +k8s:openapi-gen=true
type Port struct {
	Name string `json:"name"`
	Port int32  `json:"port"`
}

// ZookeeperSpec defines the desired state of ZookeeperSpec
// +k8s:openapi-gen=true
type ZookeeperSpec struct {
	Host string `json:"host"`
	Port Port   `json:"port"`
}

// KafkaOptions defines the desired state of KafkaOptions
// +k8s:openapi-gen=true
type KafkaOptions struct {
	TopicReplicationFactor uint `json:"topicReplicationFactor"`
	JXMPort                uint `json:"jxmport"`
}

// KafkaClusterSpec defines the desired state of KafkaCluster
// +k8s:openapi-gen=true
type KafkaClusterSpec struct {
	Replicas      int32         `json:"replicas"`
	ContainerPort Port          `json:"containerPort"`
	ServicePort   Port          `json:"servicePort"`
	Storage       string        `json:"storage"`
	Options       KafkaOptions  `json:"options"`
	Zookeeper     ZookeeperSpec `json:"zookeeper"`
	Image         string        `json:"image"`
}

// KafkaClusterStatus defines the observed state of KafkaCluster
// +k8s:openapi-gen=true
type KafkaClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaCluster is the Schema for the kafkaclusters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KafkaCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaClusterSpec   `json:"spec,omitempty"`
	Status KafkaClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaClusterList contains a list of KafkaCluster
type KafkaClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaCluster{}, &KafkaClusterList{})
}
