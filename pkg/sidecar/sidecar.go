package sidecar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ricoberger/sidecar-injector/pkg/log"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	annotationInjectKey         = "sidecar-injector.ricoberger.de"
	annotationContainersKey     = "sidecar-injector.ricoberger.de/containers"
	annotationInitContainersKey = "sidecar-injector.ricoberger.de/init-containers"
	annotationVolumesKey        = "sidecar-injector.ricoberger.de/volumes"
	annotationStatusKey         = "sidecar-injector.ricoberger.de/status"
)

type Injector struct {
	Client  client.Client
	Config  *Config
	Decoder *admission.Decoder
}

func (i *Injector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := i.Decoder.Decode(req, pod)
	if err != nil {
		log.Error("Could not decode request.", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		return admission.Errored(http.StatusBadRequest, err)
	}

	if val, ok := pod.Annotations[annotationInjectKey]; !ok || val != "enabled" {
		log.Debug("No injection required.", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		return admission.Allowed("No injection required.")
	}

	if val, ok := pod.Annotations[annotationStatusKey]; ok && val == "injected" {
		log.Debug("Already injected.", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		return admission.Allowed("Already injected.")
	}

	if initContainerNames, ok := pod.Annotations[annotationInitContainersKey]; ok && initContainerNames != "" {
		for _, initContainerName := range strings.Split(initContainerNames, ",") {
			container, err := getContainer(initContainerName, i.Config.Containers)
			if err != nil {
				log.Error("Init-Container was not found.", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace), zap.String("init-container", initContainerName))
				return admission.Errored(http.StatusBadRequest, err)
			}

			container = addEnvVariables(container, pod.Annotations, i.Config.EnvironmentVariables)
			container = setResources(container, annotationInitContainersKey, pod.Annotations)
			pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
		}
	}

	if containerNames, ok := pod.Annotations[annotationContainersKey]; ok && containerNames != "" {
		for _, containerName := range strings.Split(containerNames, ",") {
			container, err := getContainer(containerName, i.Config.Containers)
			if err != nil {
				log.Error("Container was not found.", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace), zap.String("container", containerName))
				return admission.Errored(http.StatusBadRequest, err)
			}

			container = addEnvVariables(container, pod.Annotations, i.Config.EnvironmentVariables)
			container = setResources(container, annotationContainersKey, pod.Annotations)
			pod.Spec.Containers = append(pod.Spec.Containers, container)
		}
	}

	if volumeNames, ok := pod.Annotations[annotationVolumesKey]; ok && volumeNames != "" {
		for _, volumeName := range strings.Split(volumeNames, ",") {
			volume, err := getVolume(volumeName, i.Config.Volumes)
			if err != nil {
				log.Error("Volume was not found.", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace), zap.String("volume", volumeName))
				return admission.Errored(http.StatusBadRequest, err)
			}

			pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		}
	}

	pod.Annotations[annotationStatusKey] = "injected"

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Error("Could not marshal pod.", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		return admission.Errored(http.StatusInternalServerError, err)
	}

	log.Info("Inject sidecar.", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func getContainer(name string, containers []corev1.Container) (corev1.Container, error) {
	for _, container := range containers {
		if container.Name == name {
			return container, nil
		}
	}

	return corev1.Container{}, fmt.Errorf("container not found")
}

func addEnvVariables(container corev1.Container, annotations map[string]string, environmentVariables []EnvironmentVariable) corev1.Container {
	for _, envVar := range environmentVariables {
		if envVar.Container == container.Name {
			if val, ok := annotations[envVar.Annotation]; ok && val != "" {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  envVar.Name,
					Value: val,
				})
			}
		}
	}

	return container
}

func setResources(container corev1.Container, annotationKey string, annotations map[string]string) corev1.Container {
	cpuRequestsAnnotation := fmt.Sprintf("%s/%s/%s", annotationKey, container.Name, "cpurequests")
	cpuLimitsAnnotation := fmt.Sprintf("%s/%s/%s", annotationKey, container.Name, "cpulimits")
	memoryRequestsAnnotation := fmt.Sprintf("%s/%s/%s", annotationKey, container.Name, "memoryrequests")
	memoryLimitsAnnotation := fmt.Sprintf("%s/%s/%s", annotationKey, container.Name, "memorylimits")

	if val, ok := annotations[cpuRequestsAnnotation]; ok && val != "" {
		if quantity, err := resource.ParseQuantity(val); err == nil {
			container.Resources.Requests["cpu"] = quantity
		} else {
			log.Error("Could not parse cpu requests.", zap.String("containerName", container.Name), zap.String("annotation", cpuRequestsAnnotation), zap.String("value", val), zap.Error(err))
		}
	}

	if val, ok := annotations[cpuLimitsAnnotation]; ok && val != "" {
		if quantity, err := resource.ParseQuantity(val); err == nil {
			container.Resources.Limits["cpu"] = quantity
		} else {
			log.Error("Could not parse cpu limits.", zap.String("containerName", container.Name), zap.String("annotation", cpuLimitsAnnotation), zap.String("value", val), zap.Error(err))
		}
	}

	if val, ok := annotations[memoryRequestsAnnotation]; ok && val != "" {
		if quantity, err := resource.ParseQuantity(val); err == nil {
			container.Resources.Requests["memory"] = quantity
		} else {
			log.Error("Could not parse memory requests.", zap.String("containerName", container.Name), zap.String("annotation", memoryRequestsAnnotation), zap.String("value", val), zap.Error(err))
		}
	}

	if val, ok := annotations[memoryLimitsAnnotation]; ok && val != "" {
		if quantity, err := resource.ParseQuantity(val); err == nil {
			container.Resources.Limits["memory"] = quantity
		} else {
			log.Error("Could not parse memory limits.", zap.String("containerName", container.Name), zap.String("annotation", memoryLimitsAnnotation), zap.String("value", val), zap.Error(err))
		}
	}

	return container
}

func getVolume(name string, volumes []corev1.Volume) (corev1.Volume, error) {
	for _, container := range volumes {
		if container.Name == name {
			return container, nil
		}
	}

	return corev1.Volume{}, fmt.Errorf("volume not found")
}
