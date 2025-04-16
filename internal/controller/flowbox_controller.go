package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	devflowv1 "flowbox/api/v1"
)

// FlowBoxReconciler reconciles a FlowBox object
type FlowBoxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

/*
rbac 注解
*/

// +kubebuilder:rbac:groups=devflow.flowbox.io,resources=flowboxes,verbs=*
// +kubebuilder:rbac:groups=devflow.flowbox.io,resources=flowboxes/status,verbs=*
// +kubebuilder:rbac:groups=devflow.flowbox.io,resources=flowboxes/finalizers,verbs=*
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=*
// +kubebuilder:rbac:groups=core,resources=services,verbs=*
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=*
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=*

func (r *FlowBoxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling flowbox")
	// TODO(user): your logic here

	flowbox := &devflowv1.FlowBox{}
	if err := r.Get(ctx, req.NamespacedName, flowbox); err != nil {
		if apierrors.IsNotFound(err) {
			// FlowBox 被删了，无需处理
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch FlowBox")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/*
		deployment
	*/
	desiredDeployment := deploymentCreateSpec(flowbox)
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, desiredDeployment, func() error {
		if err := ctrl.SetControllerReference(flowbox, desiredDeployment, r.Scheme); err != nil {
			return err
		}
		/*
			更新处理
		*/
		desiredDeployment.Spec.Template.Spec.Containers[0].Image = flowbox.Spec.Image
		desiredDeployment.Spec.Replicas = &flowbox.Spec.Replicas
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	/*
		service
	*/
	desiredService := serviceCreateSpec(flowbox)
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, desiredService, func() error {
		if err := ctrl.SetControllerReference(flowbox, desiredService, r.Scheme); err != nil {
			return err
		}
		/*
			更新处理
		*/
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Ingress
	*/
	desiredIngress := ingressCreateSpec(flowbox)
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, desiredIngress, func() error {
		if err := ctrl.SetControllerReference(flowbox, desiredIngress, r.Scheme); err != nil {
			return err
		}
		/*
			更新处理
		*/
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	/*
		HPA
	*/
	desiredHpa := hpaCreateSpec(flowbox)
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, desiredHpa, func() error {
		if err := ctrl.SetControllerReference(flowbox, desiredHpa, r.Scheme); err != nil {
			return err
		}
		/*
			更新处理
		*/
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

/*
Kubernetes 版本为 v1.18.6
(Ingress apiVersion 可能有不同)
flowbox 子资源需要打上 "app.kubernetes.io/name": "flowbox" 和 "app.kubernetes.io/managed-by": "flowbox",  标签
Selector 使用标签 "flowbox": flowbox.Name 标签
*/

func deploymentCreateSpec(flowbox *devflowv1.FlowBox) *appsv1.Deployment {

	// container 配置
	container := corev1.Container{
		Name:            flowbox.Name,
		Image:           flowbox.Spec.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
	}

	// 	资源限制
	if flowbox.Spec.Resource.Enabled {
		container.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", flowbox.Spec.Resource.CPU)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", flowbox.Spec.Resource.Memory)),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", flowbox.Spec.Resource.CPU)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", flowbox.Spec.Resource.Memory)),
			},
		}
	}

	// 存活探针 与 就绪探针
	if flowbox.Spec.Probe {
		container.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/" + flowbox.Name + "/actuator/health",
					Port: intstr.FromInt(int(flowbox.Spec.Port)),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       10,
		}
		container.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/" + flowbox.Name + "/actuator/health",
					Port: intstr.FromInt(int(flowbox.Spec.Port)),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       5,
		}
	}

	// deployment 配置
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowbox.Name,
			Namespace: flowbox.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "flowbox",
				"app.kubernetes.io/managed-by": "flowbox",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"flowbox": flowbox.Name,
				},
			},
			Replicas: &flowbox.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":       "flowbox",
						"app.kubernetes.io/managed-by": "flowbox",
						"flowbox":                      flowbox.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}
}

func serviceCreateSpec(flowbox *devflowv1.FlowBox) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowbox.Name,
			Namespace: flowbox.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "flowbox",
				"app.kubernetes.io/managed-by": "flowbox",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"flowbox": flowbox.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       flowbox.Name,
					Port:       flowbox.Spec.Port,
					TargetPort: intstr.FromInt(int(flowbox.Spec.Port)),
				},
			},
		},
	}
}

func ingressCreateSpec(flowbox *devflowv1.FlowBox) *networkingv1beta1.Ingress {
	if !flowbox.Spec.Ingress.Enabled {
		return nil
	}
	return &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowbox.Name,
			Namespace: flowbox.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "flowbox",
				"app.kubernetes.io/managed-by": "flowbox",
			},
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{
				{
					Host: flowbox.Spec.Ingress.Domain,
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: func() *networkingv1beta1.PathType { pt := networkingv1beta1.PathTypePrefix; return &pt }(),
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: flowbox.Name,
										ServicePort: intstr.FromInt(int(flowbox.Spec.Port)),
									},
								},
							},
						},
					},
				},
			},
		},
	}

}

func hpaCreateSpec(flowbox *devflowv1.FlowBox) *autoscalingv2beta2.HorizontalPodAutoscaler {
	if !flowbox.Spec.HPA.Enabled {
		return nil
	}
	return &autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowbox.Name,
			Namespace: flowbox.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "flowbox",
				"app.kubernetes.io/managed-by": "flowbox",
			},
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       flowbox.Name,
			},
			MinReplicas: &flowbox.Spec.HPA.Min,
			MaxReplicas: flowbox.Spec.HPA.Max,
			Metrics: []autoscalingv2beta2.MetricSpec{
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: &flowbox.Spec.HPA.CpuQuantity,
						},
					},
				},
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: &flowbox.Spec.HPA.MemoryQuantity,
						},
					},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowBoxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devflowv1.FlowBox{}).
		Named("flowbox").
		Complete(r)
}
