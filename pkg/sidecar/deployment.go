package sidecar

import (
	"context"
	"fmt"

	"github.com/tjololo/dsd/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

type Helper struct {
	Client *kubernetes.Clientset
}

func (h *Helper) AddDebugSidecar(ctx context.Context, namespace, deploymentname, containerToDebug, debugimage string) (*appsv1.Deployment, error) {
	d, err := h.Client.AppsV1().Deployments(namespace).Get(ctx, deploymentname, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	debugSidecarName := getDebugContainerName(d.Spec.Template.Spec.Containers)
	existingVolume, tmpVolume, err := getTmpVolume(d, containerToDebug)
	if err != nil {
		return nil, err
	}
	debugSidecar := generateSidecarContainerSpec(debugSidecarName, tmpVolume.Name, debugimage)
	d.Spec.Template.Spec.ShareProcessNamespace = util.BoolPtr(true)
	if !existingVolume {
		if len(d.Spec.Template.Spec.Containers) > 1 {
			for _, c := range d.Spec.Template.Spec.Containers {
				if c.Name == containerToDebug {
					c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
						Name:      tmpVolume.Name,
						MountPath: "/tmp",
					})
				}
			}
		} else {
			d.Spec.Template.Spec.Containers[0].VolumeMounts = append(d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      tmpVolume.Name,
				MountPath: "/tmp",
			})
			d.Spec.Template.Spec.Containers[0].Env = append(d.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: ""})
		}
		d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, tmpVolume)
	}
	d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, debugSidecar)
	return h.Client.AppsV1().Deployments(namespace).Update(ctx, d, metav1.UpdateOptions{})
	// return d, nil
}

func generateSidecarContainerSpec(containername, mountname, debugimage string) corev1.Container {
	return corev1.Container{
		Name:                     containername,
		Image:                    debugimage,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TTY:                      true,
		Stdin:                    true,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					corev1.Capability("SYS_PTRACE"),
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      mountname,
				MountPath: "/tmp",
			},
		},
	}
}

func getDebugContainerName(existingContainers []corev1.Container) string {
	if !debugContainerNameTaken(existingContainers) {
		return "debug"
	}
	return fmt.Sprintf("%s-%s", "debug", utilrand.String(5))
}

func debugContainerNameTaken(containers []corev1.Container) bool {
	for _, c := range containers {
		if c.Name == "debug" {
			return true
		}
	}
	return false
}

func getTmpVolume(d *appsv1.Deployment, containerToDebug string) (bool, corev1.Volume, error) {
	containerCount := len(d.Spec.Template.Spec.Containers)
	if containerCount > 1 && containerToDebug == "" {
		return false, corev1.Volume{}, fmt.Errorf("multiple containers present, pleas supply the one you want to debug")
	}
	var volumeMounts []corev1.VolumeMount
	if containerCount > 1 {
		found := false
		for _, c := range d.Spec.Template.Spec.Containers {
			if c.Name == containerToDebug {
				found = true
				volumeMounts = c.VolumeMounts
			}
		}
		if !found {
			return false, corev1.Volume{}, fmt.Errorf("could not find container with name %s in deployment %s", containerToDebug, d.Name)
		}
	} else {
		volumeMounts = d.Spec.Template.Spec.Containers[0].VolumeMounts
	}
	for _, vm := range volumeMounts {
		if vm.MountPath == "/tmp" {
			for _, v := range d.Spec.Template.Spec.Volumes {
				if v.Name == vm.Name {
					return true, v, nil
				}
			}
		}
	}
	return false, corev1.Volume{
		Name: fmt.Sprintf("tmpfolder-%s", utilrand.String(5)),
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}, nil
}
