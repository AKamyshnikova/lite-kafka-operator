package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
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
	Port *Port  `json:"port"`
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
	Replicas         int32            `json:"replicas"`
	ContainerPort    *Port            `json:"containerPort"`
	ServicePort      *Port            `json:"servicePort"`
	Storage          string           `json:"storage"`
	Options          *KafkaOptions    `json:"options"`
	Zookeeper        *ZookeeperSpec   `json:"zookeeper"`
	ZookeeperCheck   *bool            `json:"zookeeperCheck"`
	Image            string           `json:"image"`
	DataStorageClass string           `json:"dataStorageClass,omitempty"`
	Affinity         *corev1.Affinity `json:"affinity,omitempty"`
}

// KafkaClusterStatus defines the observed state of KafkaCluster
// +k8s:openapi-gen=true
type KafkaClusterStatus struct {
	ClusterStatus string `json:"clusterStatus"`
	ReadyMembers  int32  `json:"readyMembers"`
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

func (kc *KafkaCluster) GetDefaultLabels() map[string]string {
	return map[string]string{
		"app":       kc.GetName(),
		"component": "kafka-broker",
		"name":      "kafka",
	}
}

// SetDefaults Set dfault values of KafkaClusterSpec
func (kc *KafkaCluster) SetDefaults() {
	if kc.Spec.Replicas == 0 {
		kc.Spec.Replicas = 3
	}
	if len(kc.Spec.Storage) == 0 {
		kc.Spec.Storage = "1Gi"
	}
	if len(kc.Spec.Image) == 0 {
		kc.Spec.Image = "confluentinc/cp-kafka:5.0.1"
	}
	if kc.Spec.ContainerPort == nil {
		kc.Spec.ContainerPort = &Port{Name: "kafka", Port: 9092}
	}
	if kc.Spec.ServicePort == nil {
		kc.Spec.ServicePort = &Port{Name: "broker", Port: 9092}
	}
	if kc.Spec.ZookeeperCheck == nil {
		enable := true
		kc.Spec.ZookeeperCheck = &enable
	}
	if kc.Spec.Zookeeper == nil {
		kc.Spec.Zookeeper = &ZookeeperSpec{
			Host: "zookeeper",
			Port: &Port{Name: "zookeeper", Port: 2181},
		}
	} else {
		if len(kc.Spec.Zookeeper.Host) == 0 {
			kc.Spec.Zookeeper.Host = "zookeeper"
		}
		if kc.Spec.Zookeeper.Port == nil {
			kc.Spec.Zookeeper.Port = &Port{Name: "zookeeper", Port: 2181}
		}
	}
	if kc.Spec.Options == nil {
		kc.Spec.Options = &KafkaOptions{
			TopicReplicationFactor: 2,
			JXMPort:                5555,
		}
	} else {
		if kc.Spec.Options.TopicReplicationFactor == 0 {
			kc.Spec.Options.TopicReplicationFactor = 2
		}
		if kc.Spec.Options.JXMPort == 0 {
			kc.Spec.Options.JXMPort = 5555
		}
	}

	if kc.Spec.Affinity == nil {
		kc.Spec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
					{
						Weight: 20,
						PodAffinityTerm: corev1.PodAffinityTerm{
							TopologyKey: "kubernetes.io/hostname",
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "app",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{kc.GetName()},
									},
								},
							},
						},
					},
				},
			},
		}
	}
}
