// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package dogstatsd

import (
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	apicommon "github.com/DataDog/datadog-operator/api/datadoghq/common"
	apicommonv1 "github.com/DataDog/datadog-operator/api/datadoghq/common/v1"
	"github.com/DataDog/datadog-operator/api/datadoghq/v2alpha1"
	apiutils "github.com/DataDog/datadog-operator/api/utils"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/common"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/merger"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/object/volume"
)

func init() {
	err := feature.Register(feature.DogstatsdIDType, buildDogstatsdFeature)
	if err != nil {
		panic(err)
	}
}

func buildDogstatsdFeature(options *feature.Options) feature.Feature {
	dogstatsdFeat := &dogstatsdFeature{}

	return dogstatsdFeat
}

type dogstatsdFeature struct {
	hostPortEnabled  bool
	hostPortHostPort int32

	udsEnabled      bool
	udsHostFilepath string

	useHostNetwork         bool
	originDetectionEnabled bool
	tagCardinality         string
	mapperProfiles         *apicommonv1.CustomConfig

	forceEnableLocalService bool
	localServiceName        string

	owner metav1.Object
}

// ID returns the ID of the Feature
func (f *dogstatsdFeature) ID() feature.IDType {
	return feature.DogstatsdIDType
}

// Configure is used to configure the feature from a v2alpha1.DatadogAgent instance.
func (f *dogstatsdFeature) Configure(dda *v2alpha1.DatadogAgent) (reqComp feature.RequiredComponents) {
	dogstatsd := dda.Spec.Features.Dogstatsd
	f.owner = dda
	if apiutils.BoolValue(dogstatsd.HostPortConfig.Enabled) {
		f.hostPortEnabled = true
		f.hostPortHostPort = *dogstatsd.HostPortConfig.Port
	}
	// UDS is enabled by default
	if apiutils.BoolValue(dogstatsd.UnixDomainSocketConfig.Enabled) {
		f.udsEnabled = true
	}
	f.udsHostFilepath = *dogstatsd.UnixDomainSocketConfig.Path
	if apiutils.BoolValue(dogstatsd.OriginDetectionEnabled) {
		f.originDetectionEnabled = true
	}
	if dogstatsd.TagCardinality != nil {
		f.tagCardinality = *dogstatsd.TagCardinality
	}
	f.useHostNetwork = v2alpha1.IsHostNetworkEnabled(dda, v2alpha1.NodeAgentComponentName)
	if dogstatsd.MapperProfiles != nil {
		f.mapperProfiles = v2alpha1.ConvertCustomConfig(dogstatsd.MapperProfiles)
	}

	if dda.Spec.Global.LocalService != nil {
		f.forceEnableLocalService = apiutils.BoolValue(dda.Spec.Global.LocalService.ForceEnableLocalService)
	}
	f.localServiceName = v2alpha1.GetLocalAgentServiceName(dda)

	reqComp = feature.RequiredComponents{
		Agent: feature.RequiredComponent{
			IsRequired: apiutils.NewBoolPointer(true),
			Containers: []apicommonv1.AgentContainerName{
				apicommonv1.CoreAgentContainerName,
			},
		},
	}
	return reqComp
}

// ManageDependencies allows a feature to manage its dependencies.
// Feature's dependencies should be added in the store.
func (f *dogstatsdFeature) ManageDependencies(managers feature.ResourceManagers, components feature.RequiredComponents) error {
	// agent local service
	if common.ShouldCreateAgentLocalService(managers.Store().GetVersionInfo(), f.forceEnableLocalService) {
		dsdPort := &corev1.ServicePort{
			Protocol:   corev1.ProtocolUDP,
			TargetPort: intstr.FromInt(int(apicommon.DefaultDogstatsdPort)),
			Port:       apicommon.DefaultDogstatsdPort,
			Name:       apicommon.DefaultDogstatsdPortName,
		}
		if f.hostPortEnabled {
			dsdPort.Port = f.hostPortHostPort
			dsdPort.Name = apicommon.DogstatsdHostPortName
			if f.useHostNetwork {
				dsdPort.TargetPort = intstr.FromInt(int(f.hostPortHostPort))
			}
		}
		serviceInternalTrafficPolicy := corev1.ServiceInternalTrafficPolicyLocal
		if err := managers.ServiceManager().AddService(f.localServiceName, f.owner.GetNamespace(), common.GetAgentLocalServiceSelector(f.owner), []corev1.ServicePort{*dsdPort}, &serviceInternalTrafficPolicy); err != nil {
			return err
		}
	}

	return nil
}

// ManageClusterAgent allows a feature to configure the ClusterAgent's corev1.PodTemplateSpec
// It should do nothing if the feature doesn't need to configure it.
func (f *dogstatsdFeature) ManageClusterAgent(managers feature.PodTemplateManagers) error {
	return nil
}

// ManageSingleContainerNodeAgent allows a feature to configure the Agent container for the Node Agent's corev1.PodTemplateSpec
// if SingleContainerStrategy is enabled and can be used with the configured feature set.
// It should do nothing if the feature doesn't need to configure it.
func (f *dogstatsdFeature) ManageSingleContainerNodeAgent(managers feature.PodTemplateManagers, provider string) error {
	f.manageNodeAgent(apicommonv1.UnprivilegedSingleAgentContainerName, managers, provider)
	return nil
}

// ManageNodeAgent allows a feature to configure the Node Agent's corev1.PodTemplateSpec
// It should do nothing if the feature doesn't need to configure it.
func (f *dogstatsdFeature) ManageNodeAgent(managers feature.PodTemplateManagers, provider string) error {
	f.manageNodeAgent(apicommonv1.CoreAgentContainerName, managers, provider)
	return nil
}

func (f *dogstatsdFeature) manageNodeAgent(agentContainerName apicommonv1.AgentContainerName, managers feature.PodTemplateManagers, provider string) error {
	// udp
	dogstatsdPort := &corev1.ContainerPort{
		Name:          apicommon.DefaultDogstatsdPortName,
		ContainerPort: apicommon.DefaultDogstatsdPort,
		Protocol:      corev1.ProtocolUDP,
	}
	if f.hostPortEnabled {
		// f.hostPortHostPort will be 0 if HostPort is not set in v1alpha1
		// f.hostPortHostPort will default to 8125 in v2alpha1
		dsdPortEnvVarValue := apicommon.DefaultDogstatsdPort
		if f.hostPortHostPort != 0 {
			dogstatsdPort.HostPort = f.hostPortHostPort
			// if using host network, host port should be set and needs to match container port
			if f.useHostNetwork {
				dogstatsdPort.ContainerPort = f.hostPortHostPort
				dsdPortEnvVarValue = int(f.hostPortHostPort)
			}
		}
		managers.EnvVar().AddEnvVarToContainer(agentContainerName, &corev1.EnvVar{
			// defaults to 8125 in datadog-agent code
			Name:  apicommon.DDDogstatsdPort,
			Value: strconv.Itoa(dsdPortEnvVarValue),
		})
		managers.EnvVar().AddEnvVarToContainer(agentContainerName, &corev1.EnvVar{
			Name:  apicommon.DDDogstatsdNonLocalTraffic,
			Value: "true",
		})
	}
	managers.Port().AddPortToContainer(agentContainerName, dogstatsdPort)

	// uds
	if f.udsEnabled {
		udsHostFolder := filepath.Dir(f.udsHostFilepath)
		sockName := filepath.Base(f.udsHostFilepath)
		socketVol, socketVolMount := volume.GetVolumes(apicommon.DogstatsdSocketVolumeName, udsHostFolder, apicommon.DogstatsdSocketLocalPath, false)
		volType := corev1.HostPathDirectoryOrCreate // We need to create the directory on the host if it does not exist.

		socketVol.VolumeSource.HostPath.Type = &volType
		managers.VolumeMount().AddVolumeMountToContainerWithMergeFunc(&socketVolMount, agentContainerName, merger.OverrideCurrentVolumeMountMergeFunction)
		managers.Volume().AddVolume(&socketVol)
		managers.EnvVar().AddEnvVar(&corev1.EnvVar{
			Name:  apicommon.DDDogstatsdSocket,
			Value: filepath.Join(apicommon.DogstatsdSocketLocalPath, sockName),
		})
	}

	if f.originDetectionEnabled {
		managers.EnvVar().AddEnvVarToContainer(agentContainerName, &corev1.EnvVar{
			Name:  apicommon.DDDogstatsdOriginDetection,
			Value: "true",
		})
		managers.EnvVar().AddEnvVarToContainer(agentContainerName, &corev1.EnvVar{
			Name:  apicommon.DDDogstatsdOriginDetectionClient,
			Value: "true",
		})
		if f.udsEnabled {
			managers.PodTemplateSpec().Spec.HostPID = true
		}
		// Tag cardinality is only configured if origin detection is enabled.
		// The value validation happens at the Agent level - if the lower(string) is not `low`, `orchestrator` or `high`, the Agent defaults to `low`.
		if f.tagCardinality != "" {
			managers.EnvVar().AddEnvVarToContainer(apicommonv1.CoreAgentContainerName, &corev1.EnvVar{
				Name:  apicommon.DDDogstatsdTagCardinality,
				Value: f.tagCardinality,
			})
		}
	}

	// mapper profiles
	if f.mapperProfiles != nil {
		// configdata
		if f.mapperProfiles.ConfigData != nil {
			managers.EnvVar().AddEnvVarToContainer(apicommonv1.CoreAgentContainerName, &corev1.EnvVar{
				Name:  apicommon.DDDogstatsdMapperProfiles,
				Value: apiutils.YAMLToJSONString(*f.mapperProfiles.ConfigData),
			})
			// ignore configmap if configdata is set
			return nil
		}
		// configmap
		if f.mapperProfiles.ConfigMap != nil {
			cmSelector := corev1.ConfigMapKeySelector{}
			cmSelector.Name = f.mapperProfiles.ConfigMap.Name
			cmSelector.Key = f.mapperProfiles.ConfigMap.Items[0].Key
			managers.EnvVar().AddEnvVarToContainer(apicommonv1.CoreAgentContainerName, &corev1.EnvVar{
				Name:      apicommon.DDDogstatsdMapperProfiles,
				ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &cmSelector},
			})
		}
	}

	return nil
}

// ManageClusterChecksRunner allows a feature to configure the ClusterChecksRunner's corev1.PodTemplateSpec
// It should do nothing if the feature doesn't need to configure it.
func (f *dogstatsdFeature) ManageClusterChecksRunner(managers feature.PodTemplateManagers) error {
	return nil
}
