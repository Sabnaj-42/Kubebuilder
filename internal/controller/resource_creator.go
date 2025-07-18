package controller

import (
	controllerv1 "github.com/Sabnaj-42/kubebuilder/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	logger "log"
	"strings"
)

func newDeployment(crd *controllerv1.CRD) *appsv1.Deployment {
	deploymentName := crd.Spec.Name
	if deploymentName == "" {
		deploymentName = strings.Join([]string{crd.Name, controllerv1.Deployment}, controllerv1.Dash)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: crd.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(crd, controllerv1.GroupVersion.WithKind(controllerv1.MyKind)),
			},
		},

		Spec: appsv1.DeploymentSpec{
			Replicas: crd.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					controllerv1.AppLabel:  strings.Join([]string{controllerv1.AppLabel, crd.Name}, controllerv1.Dash),
					controllerv1.NameLabel: crd.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controllerv1.AppLabel:  strings.Join([]string{controllerv1.AppLabel, crd.Name}, controllerv1.Dash),
						controllerv1.NameLabel: crd.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  controllerv1.AppValue,
							Image: crd.Spec.Container.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: crd.Spec.Container.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}

func newService(crd *controllerv1.CRD) *corev1.Service {
	serviceName := crd.Spec.Name
	logger.Println("Inside Service Creation Function")
	if serviceName == "" {
		serviceName = strings.Join([]string{crd.Name, controllerv1.Deployment}, controllerv1.Dash)
	}
	logger.Println("Service Name, ", serviceName)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: crd.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(crd, controllerv1.GroupVersion.WithKind(controllerv1.MyKind)),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				controllerv1.AppLabel:  strings.Join([]string{controllerv1.AppLabel, crd.Name}, controllerv1.Dash),
				controllerv1.NameLabel: crd.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       controllerv1.ServicePort,
					TargetPort: intstr.FromInt(int(crd.Spec.Container.Port)),
				},
			},
		},
	}

}
