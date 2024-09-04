/*
Copyright 2024.

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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	datav1alpha1 "github.com/tomer-km/test-operator.git/api/v1alpha1"
)

// DatareaderReconciler reconciles a Datareader object
type DatareaderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func GetDeploymentSpec(NamespacedName types.NamespacedName, volumespecs []apiv1.Volume, volumemounts []apiv1.VolumeMount, image string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacedName.Name,
			Namespace: NamespacedName.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"reader": NamespacedName.Name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"reader": NamespacedName.Name,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "reader",
							Image: image,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
							VolumeMounts: volumemounts,
						},
					},
					Volumes: volumespecs,
				},
			},
		},
	}
	return deployment

}

func int32Ptr(i int32) *int32 { return &i }

//+kubebuilder:rbac:groups=data.tomer.com,resources=datareaders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=data.tomer.com,resources=datareaders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=data.tomer.com,resources=datareaders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Datareader object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *DatareaderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	//logger := log.WithValues("service", req.NamespacedName)
	reader := &datav1alpha1.Datareader{}
	err := r.Get(ctx, req.NamespacedName, reader)
	if err != nil {
		return ctrl.Result{}, nil
	}

	configmaps := &apiv1.ConfigMapList{}
	deployment := &appsv1.Deployment{}

	selector := labels.SelectorFromSet(map[string]string{"reader": "true"})

	err = r.List(ctx, configmaps, &client.ListOptions{LabelSelector: selector, Namespace: req.NamespacedName.Namespace})
	if err != nil {
		panic(err.Error())
	}

	var volumespecs []apiv1.Volume
	var volumemounts []apiv1.VolumeMount

	for _, configmap := range configmaps.Items {
		volumespecs = append(volumespecs, apiv1.Volume{Name: configmap.Name, VolumeSource: apiv1.VolumeSource{ConfigMap: &apiv1.ConfigMapVolumeSource{LocalObjectReference: apiv1.LocalObjectReference{Name: configmap.Name}}}})
		volumemounts = append(volumemounts, apiv1.VolumeMount{Name: configmap.Name, MountPath: "/tmp/cm/" + configmap.Name})
	}

	image := func(image string) string {

		if len(image) == 0 {
			return "nginx:1.12"

		}

		return image
	}(reader.Spec.Image)

	deploymentSpec := GetDeploymentSpec(req.NamespacedName, volumespecs, volumemounts, image)

	err = r.Get(ctx, req.NamespacedName, deployment, &client.GetOptions{})
	if err != nil {

		err = r.Create(ctx, deploymentSpec, &client.CreateOptions{})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Created deployment.\n")
	} else {
		err = r.Update(ctx, deploymentSpec, &client.UpdateOptions{})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Deployment Reconciled.\n")

	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatareaderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1alpha1.Datareader{}).
		Complete(r)
}
