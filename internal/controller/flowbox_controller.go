/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	devflowv1 "flowbox/api/v1"
)

// FlowBoxReconciler reconciles a FlowBox object
type FlowBoxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=devflow.flowbox.io,resources=flowboxes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=devflow.flowbox.io,resources=flowboxes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=devflow.flowbox.io,resources=flowboxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FlowBox object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *FlowBoxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling flowbox")
	// TODO(user): your logic here

	// 获取 FlowBox CR
	flowbox := &devflowv1.FlowBox{}
	if err := r.Get(ctx, req.NamespacedName, flowbox); err != nil {
		if apierrors.IsNotFound(err) {
			// FlowBox 被删了，无需处理
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch FlowBox")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 构造期望的 Deployment Service Ingress 资源
	desiredDeployment := deploymentCreateSpec(flowbox)
	desiredService := serviceCreateSpec(flowbox)
	desiredIngress := ingressCreateSpec(flowbox)

	// 尝试获取现有 Deployment
	currentDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      desiredDeployment.Name,
		Namespace: desiredDeployment.Namespace,
	}, currentDeployment)

	if err != nil && apierrors.IsNotFound(err) {

		// deployment
		if err = ctrl.SetControllerReference(flowbox, desiredDeployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, desiredDeployment); err != nil {
			log.Error(err, "unable to create Deployment: "+err.Error())
			return ctrl.Result{}, err
		}

		// service
		if err = ctrl.SetControllerReference(flowbox, desiredService, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, desiredService); err != nil {
			log.Error(err, "unable to create Service: "+err.Error())
			return ctrl.Result{}, err
		}

		// Ingress
		if desiredIngress != nil {
			if err = ctrl.SetControllerReference(flowbox, desiredIngress, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err = r.Create(ctx, desiredIngress); err != nil {
				log.Error(err, "unable to create ingress: "+err.Error())
				return ctrl.Result{}, err
			}
		}

		log.Info("flowbox process created")
	} else if err == nil {
		// 如果存在，可以在这里判断差异，决定是否更新（推荐后面用 CreateOrUpdate 简化）
		log.Info("Deployment already exists")
	} else {
		// 其他错误
		log.Error(err, "unable to get Deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func deploymentCreateSpec(flowbox *devflowv1.FlowBox) *appsv1.Deployment {
	container := corev1.Container{
		Name:  flowbox.Name,
		Image: flowbox.Spec.Image,
	}

	if flowbox.Spec.HealthProbe {
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

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowbox.Name,
			Namespace: flowbox.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": flowbox.Name,
				},
			},
			Replicas: &flowbox.Spec.ReplicaCount,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": flowbox.Name,
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
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": flowbox.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       flowbox.Name,
					Port:       int32(flowbox.Spec.Port),
					TargetPort: intstr.FromInt(int(flowbox.Spec.Port)),
				},
			},
		},
	}
}

func ingressCreateSpec(flowbox *devflowv1.FlowBox) *networkingv1beta1.Ingress {
	if flowbox.Spec.FlowBoxIngressSpec.Enabled {
		return &networkingv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      flowbox.Name,
				Namespace: flowbox.Namespace,
			},
			Spec: networkingv1beta1.IngressSpec{
				Rules: []networkingv1beta1.IngressRule{
					{
						Host: flowbox.Spec.FlowBoxIngressSpec.Domain,
						IngressRuleValue: networkingv1beta1.IngressRuleValue{
							HTTP: &networkingv1beta1.HTTPIngressRuleValue{
								Paths: []networkingv1beta1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: func() *networkingv1beta1.PathType { pt := networkingv1beta1.PathTypePrefix; return &pt }(),
										Backend: networkingv1beta1.IngressBackend{
											ServiceName: flowbox.Spec.Name,
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
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowBoxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devflowv1.FlowBox{}).
		Named("flowbox").
		Complete(r)
}
