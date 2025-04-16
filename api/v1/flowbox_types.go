package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FlowBoxSpec defines the desired state of FlowBox.
type FlowBoxSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// flowbox 应用端口
	Port int32 `json:"port,omitempty"`

	// flowbox 应用镜像
	Image string `json:"image,omitempty"`

	// flowbox pod 副本数
	Replicas int32 `json:"replicas,omitempty"`

	// flowbox 应用健康检查是否开启，默认uri /<flowbox_name>/actuator/health
	Probe bool `json:"probe,omitempty"`

	HPA HpaSpec `json:"hpa,omitempty"`

	// flowbox 应用资源限制开关
	Resource ResourceSpec `json:"resource,omitempty"`

	// Ingress 配置
	Ingress IngressSpec `json:"ingress,omitempty"`
}

type HpaSpec struct {
	// 是否开启HPA
	Enabled bool `json:"enabled,omitempty"`

	// 最小数
	Min int32 `json:"min,omitempty"`

	// 最大数
	Max int32 `json:"max,omitempty"`

	// CPU 阀值
	CpuQuantity int32 `json:"cpuQuantity,omitempty"`

	// Memory 阀值
	MemoryQuantity int32 `json:"memoryQuantity,omitempty"`
}

type ResourceSpec struct {
	// 是否开启资源限制
	Enabled bool `json:"enabled,omitempty"`

	// CPU 单位 m; 1C=1000m
	CPU int `json:"cpu,omitempty"`

	// Memory 单位 Mi; 1G = 1024Mi
	Memory int `json:"memory,omitempty"`
}

type IngressSpec struct {
	// 是否启用 Ingress
	Enabled bool `json:"enabled,omitempty"`

	// 域名配置
	Domain string `json:"domain,omitempty"`
}

// FlowBoxStatus defines the observed state of FlowBox.
type FlowBoxStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FlowBox is the Schema for the flowboxes API.
type FlowBox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowBoxSpec   `json:"spec,omitempty"`
	Status FlowBoxStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlowBoxList contains a list of FlowBox.
type FlowBoxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlowBox `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FlowBox{}, &FlowBoxList{})
}
