package sidecar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	annotationInjectKey         = "sidecar-injector.ricoberger.de"
	annotationContainersKey     = "sidecar-injector.ricoberger.de/containers"
	annotationInitContainersKey = "sidecar-injector.ricoberger.de/init-containers"
	annotationVolumesKey        = "sidecar-injector.ricoberger.de/volumes"
	annotationStatusKey         = "sidecar-injector.ricoberger.de/status"
)

var (
	log = logf.Log.WithName("sidecar")
)

type Injector struct {
	Client  client.Client
	Config  *Config
	Decoder admission.Decoder
}

func (i *Injector) getResourcesToInject(req admission.Request, pod *corev1.Pod) ([]string, []string, []string, bool, error) {
	var initContainers []string
	var containers []string
	var volumes []string

	// If the Pod already has the annotation
	// `sidecar-injector.ricoberger.de/status` set to `injected` we can skip the
	// injection of resources, because this was already done.
	if val, ok := pod.Annotations[annotationStatusKey]; ok && val == "injected" {
		log.Info("Already injected.", "name", req.Name, "namespace", req.Namespace)
		return nil, nil, nil, false, nil
	}

	// Check if the Pod matches an defined injector, by comparing the labels of
	// the Pod with the defined selector of the injector definition. If the Pod
	// matches the selector we add the defined resources in the injector to the
	// list of resources which should be injected.
	for _, injector := range i.Config.Injectors {
		selector, err := metav1.LabelSelectorAsSelector(&injector.Selector)
		if err != nil {
			log.Error(err, "Failed to convert label selector to selector.", "name", req.Name, "namespace", req.Namespace)
			return nil, nil, nil, false, err
		}

		if selector.Matches(labels.Set(pod.Labels)) {
			initContainers = injector.InitContainers
			containers = injector.Containers
			volumes = injector.Volumes
		}
	}

	// Check if the Pod has the `sidecar-injector.ricoberger.de` annotation,
	// which means that the resources which should be injected are defined
	// within the annotations of the Pod.
	//
	// If the Pod doesn't have the label and didn't matched any of the defined
	// injectors from the config, we can skip the injection of sidecars.
	if val, ok := pod.Annotations[annotationInjectKey]; (!ok || val != "enabled") && (len(initContainers) == 0 && len(containers) == 0 && len(volumes) == 0) {
		log.Info("No injection required.", "name", req.Name, "namespace", req.Namespace)
		return nil, nil, nil, false, nil
	}

	// Check the sidecar injector annotations of the Pod and add the defined
	// Init Containers, Containers and Volumes to the lists of resources, which
	// should be injected into the Pod.
	if initContainerNames, ok := pod.Annotations[annotationInitContainersKey]; ok && initContainerNames != "" {
		initContainers = append(initContainers, strings.Split(initContainerNames, ",")...)
	}

	if containerNames, ok := pod.Annotations[annotationContainersKey]; ok && containerNames != "" {
		containers = append(containers, strings.Split(containerNames, ",")...)
	}

	if volumeNames, ok := pod.Annotations[annotationVolumesKey]; ok && volumeNames != "" {
		volumes = append(volumes, strings.Split(volumeNames, ",")...)
	}

	return initContainers, containers, volumes, true, nil
}

func (i *Injector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := i.Decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "Could not decode request.", "name", req.Name, "namespace", req.Namespace)
		return admission.Errored(http.StatusBadRequest, err)
	}

	initContainers, containers, volumes, inject, err := i.getResourcesToInject(req, pod)
	if err != nil {
		log.Error(err, "Failed to get resources to inject.", "name", req.Name, "namespace", req.Namespace)
		return admission.Errored(http.StatusBadRequest, err)
	}
	if !inject {
		return admission.Allowed("No injection required.")
	}

	for _, initContainerName := range initContainers {
		container, err := getContainer(initContainerName, i.Config.Containers)
		if err != nil {
			log.Error(err, "Init-Container was not found.", "name", req.Name, "namespace", req.Namespace, "init-container", initContainerName)
			return admission.Errored(http.StatusBadRequest, err)
		}

		container = addEnvVariables(container, pod.Annotations, i.Config.EnvironmentVariables)
		container = setResources(container, annotationInitContainersKey, pod.Annotations)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	}

	for _, containerName := range containers {
		container, err := getContainer(containerName, i.Config.Containers)
		if err != nil {
			log.Error(err, "Container was not found.", "name", req.Name, "namespace", req.Namespace, "container", containerName)
			return admission.Errored(http.StatusBadRequest, err)
		}

		container = addEnvVariables(container, pod.Annotations, i.Config.EnvironmentVariables)
		container = setResources(container, annotationContainersKey, pod.Annotations)
		pod.Spec.Containers = append(pod.Spec.Containers, container)
	}

	for _, volumeName := range volumes {
		volume, err := getVolume(volumeName, i.Config.Volumes)
		if err != nil {
			log.Error(err, "Volume was not found.", "name", req.Name, "namespace", req.Namespace, "volume", volumeName)
			return admission.Errored(http.StatusBadRequest, err)
		}

		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{annotationStatusKey: "injected"}
	} else {
		pod.Annotations[annotationStatusKey] = "injected"
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Error(err, "Could not marshal pod.", "name", req.Name, "namespace", req.Namespace)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	log.Info("Inject sidecar.", "name", req.Name, "namespace", req.Namespace)
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func getContainer(name string, containers []corev1.Container) (corev1.Container, error) {
	for _, container := range containers {
		if container.Name == name {
			return *container.DeepCopy(), nil
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
	cpuRequestsAnnotation := fmt.Sprintf("%s-%s-%s", annotationKey, container.Name, "cpurequests")
	cpuLimitsAnnotation := fmt.Sprintf("%s-%s-%s", annotationKey, container.Name, "cpulimits")
	memoryRequestsAnnotation := fmt.Sprintf("%s-%s-%s", annotationKey, container.Name, "memoryrequests")
	memoryLimitsAnnotation := fmt.Sprintf("%s-%s-%s", annotationKey, container.Name, "memorylimits")

	if val, ok := annotations[cpuRequestsAnnotation]; ok && val != "" {
		quantity, err := resource.ParseQuantity(val)
		if err != nil {
			log.Error(err, "Could not parse cpu requests.", "containerName", container.Name, "annotation", cpuRequestsAnnotation, "value", val)
		} else {
			container.Resources.Requests["cpu"] = quantity
		}
	}

	if val, ok := annotations[cpuLimitsAnnotation]; ok && val != "" {
		quantity, err := resource.ParseQuantity(val)
		if err != nil {
			log.Error(err, "Could not parse cpu limits.", "containerName", container.Name, "annotation", cpuLimitsAnnotation, "value", val)
		} else {
			container.Resources.Limits["cpu"] = quantity
		}
	}

	if val, ok := annotations[memoryRequestsAnnotation]; ok && val != "" {
		quantity, err := resource.ParseQuantity(val)
		if err != nil {
			log.Error(err, "Could not parse memory requests.", "containerName", container.Name, "annotation", memoryRequestsAnnotation, "value", val)
		} else {
			container.Resources.Requests["memory"] = quantity
		}
	}

	if val, ok := annotations[memoryLimitsAnnotation]; ok && val != "" {
		quantity, err := resource.ParseQuantity(val)
		if err != nil {
			log.Error(err, "Could not parse memory limits.", "containerName", container.Name, "annotation", memoryLimitsAnnotation, "value", val)
		} else {
			container.Resources.Limits["memory"] = quantity
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
