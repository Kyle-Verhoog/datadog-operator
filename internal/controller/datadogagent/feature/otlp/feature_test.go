// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package otlp

import (
	"testing"

	apicommon "github.com/DataDog/datadog-operator/api/datadoghq/common"
	apicommonv1 "github.com/DataDog/datadog-operator/api/datadoghq/common/v1"
	"github.com/DataDog/datadog-operator/api/datadoghq/v2alpha1"
	v2alpha1test "github.com/DataDog/datadog-operator/api/datadoghq/v2alpha1/test"
	apiutils "github.com/DataDog/datadog-operator/api/utils"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature/fake"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature/test"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestOTLPFeature(t *testing.T) {
	tests := test.FeatureTestSuite{
		{
			Name: "gRPC and HTTP enabled, APM",
			DDA: newAgent(Settings{
				EnabledGRPC:  true,
				EndpointGRPC: "0.0.0.0:4317",
				EnabledHTTP:  true,
				EndpointHTTP: "0.0.0.0:4318",
				APM:          true,
			}),
			WantConfigure: true,
			Agent: testExpected(Expected{
				EnvVars: []*corev1.EnvVar{
					{
						Name:  apicommon.DDOTLPgRPCEndpoint,
						Value: "0.0.0.0:4317",
					},
					{
						Name:  apicommon.DDOTLPHTTPEndpoint,
						Value: "0.0.0.0:4318",
					},
				},
				CheckTraceAgent: true,
				Ports: []*corev1.ContainerPort{
					{
						Name:          apicommon.OTLPGRPCPortName,
						ContainerPort: 4317,
						HostPort:      4317,
						Protocol:      corev1.ProtocolTCP,
					},
					{
						Name:          apicommon.OTLPHTTPPortName,
						ContainerPort: 4318,
						HostPort:      4318,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			}),
		},
		{
			Name: "[single container] gRPC and HTTP enabled, APM",
			DDA: newAgentSingleContainer(Settings{
				EnabledGRPC:  true,
				EndpointGRPC: "0.0.0.0:4317",
				EnabledHTTP:  true,
				EndpointHTTP: "0.0.0.0:4318",
				APM:          true,
			}),
			WantConfigure: true,
			Agent: testExpectedSingleContainer(Expected{
				EnvVars: []*corev1.EnvVar{
					{
						Name:  apicommon.DDOTLPgRPCEndpoint,
						Value: "0.0.0.0:4317",
					},
					{
						Name:  apicommon.DDOTLPHTTPEndpoint,
						Value: "0.0.0.0:4318",
					},
				},
				CheckTraceAgent: true,
				Ports: []*corev1.ContainerPort{
					{
						Name:          apicommon.OTLPGRPCPortName,
						ContainerPort: 4317,
						HostPort:      4317,
						Protocol:      corev1.ProtocolTCP,
					},
					{
						Name:          apicommon.OTLPHTTPPortName,
						ContainerPort: 4318,
						HostPort:      4318,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			}),
		},
		{
			Name: "gRPC enabled, no APM",
			DDA: newAgent(Settings{
				EnabledGRPC:  true,
				EndpointGRPC: "0.0.0.0:4317",
			}),
			WantConfigure: true,
			Agent: testExpected(Expected{
				EnvVars: []*corev1.EnvVar{
					{
						Name:  apicommon.DDOTLPgRPCEndpoint,
						Value: "0.0.0.0:4317",
					},
				},
				Ports: []*corev1.ContainerPort{
					{
						Name:          apicommon.OTLPGRPCPortName,
						ContainerPort: 4317,
						HostPort:      4317,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			}),
		},
		{
			Name: "[single container] gRPC enabled, no APM",
			DDA: newAgentSingleContainer(Settings{
				EnabledGRPC:  true,
				EndpointGRPC: "0.0.0.0:4317",
			}),
			WantConfigure: true,
			Agent: testExpectedSingleContainer(Expected{
				EnvVars: []*corev1.EnvVar{
					{
						Name:  apicommon.DDOTLPgRPCEndpoint,
						Value: "0.0.0.0:4317",
					},
				},
				Ports: []*corev1.ContainerPort{
					{
						Name:          apicommon.OTLPGRPCPortName,
						ContainerPort: 4317,
						HostPort:      4317,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			}),
		},
		{
			Name: "HTTP enabled, APM",
			DDA: newAgent(Settings{
				EnabledHTTP:  true,
				EndpointHTTP: "somehostname:4318",
				APM:          true,
			}),
			WantConfigure: true,
			Agent: testExpected(Expected{
				EnvVars: []*corev1.EnvVar{
					{
						Name:  apicommon.DDOTLPHTTPEndpoint,
						Value: "somehostname:4318",
					},
				},
				CheckTraceAgent: true,
				Ports: []*corev1.ContainerPort{
					{
						Name:          apicommon.OTLPHTTPPortName,
						ContainerPort: 4318,
						HostPort:      4318,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			}),
		},
		{
			Name: "[single container] HTTP enabled, APM",
			DDA: newAgentSingleContainer(Settings{
				EnabledHTTP:  true,
				EndpointHTTP: "somehostname:4318",
				APM:          true,
			}),
			WantConfigure: true,
			Agent: testExpectedSingleContainer(Expected{
				EnvVars: []*corev1.EnvVar{
					{
						Name:  apicommon.DDOTLPHTTPEndpoint,
						Value: "somehostname:4318",
					},
				},
				CheckTraceAgent: true,
				Ports: []*corev1.ContainerPort{
					{
						Name:          apicommon.OTLPHTTPPortName,
						ContainerPort: 4318,
						HostPort:      4318,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			}),
		},
	}

	tests.Run(t, buildOTLPFeature)
}

type Settings struct {
	EnabledGRPC  bool
	EndpointGRPC string
	EnabledHTTP  bool
	EndpointHTTP string

	APM bool
}

func newAgent(set Settings) *v2alpha1.DatadogAgent {
	return v2alpha1test.NewDatadogAgentBuilder().
		WithOTLPGRPCSettings(set.EnabledGRPC, set.EndpointGRPC).
		WithOTLPHTTPSettings(set.EnabledHTTP, set.EndpointHTTP).
		WithAPMEnabled(set.APM).
		Build()
}

func newAgentSingleContainer(set Settings) *v2alpha1.DatadogAgent {
	return v2alpha1test.NewDatadogAgentBuilder().
		WithOTLPGRPCSettings(set.EnabledGRPC, set.EndpointGRPC).
		WithOTLPHTTPSettings(set.EnabledHTTP, set.EndpointHTTP).
		WithAPMEnabled(set.APM).
		WithSingleContainerStrategy(true).
		Build()
}

type Expected struct {
	EnvVars         []*corev1.EnvVar
	CheckTraceAgent bool
	Ports           []*corev1.ContainerPort
}

func testExpected(exp Expected) *test.ComponentTest {
	return test.NewDefaultComponentTest().WithWantFunc(
		func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
			mgr := mgrInterface.(*fake.PodTemplateManagers)

			agentEnvs := mgr.EnvVarMgr.EnvVarsByC[apicommonv1.CoreAgentContainerName]
			assert.True(
				t,
				apiutils.IsEqualStruct(agentEnvs, exp.EnvVars),
				"Core Agent ENVs \ndiff = %s", cmp.Diff(agentEnvs, exp.EnvVars),
			)

			if exp.CheckTraceAgent {
				agentEnvs := mgr.EnvVarMgr.EnvVarsByC[apicommonv1.TraceAgentContainerName]
				assert.True(
					t,
					apiutils.IsEqualStruct(agentEnvs, exp.EnvVars),
					"Trace Agent ENVs \ndiff = %s", cmp.Diff(agentEnvs, exp.EnvVars),
				)
			}

			agentPorts := mgr.PortMgr.PortsByC[apicommonv1.CoreAgentContainerName]
			assert.True(
				t,
				apiutils.IsEqualStruct(agentPorts, exp.Ports),
				"Core Agent Ports \ndiff = %s", cmp.Diff(agentPorts, exp.Ports),
			)
		},
	)
}

func testExpectedSingleContainer(exp Expected) *test.ComponentTest {
	return test.NewDefaultComponentTest().WithWantFunc(
		func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
			mgr := mgrInterface.(*fake.PodTemplateManagers)

			agentEnvs := mgr.EnvVarMgr.EnvVarsByC[apicommonv1.UnprivilegedSingleAgentContainerName]
			assert.True(
				t,
				apiutils.IsEqualStruct(agentEnvs, exp.EnvVars),
				"Core Agent ENVs \ndiff = %s", cmp.Diff(agentEnvs, exp.EnvVars),
			)

			if exp.CheckTraceAgent {
				agentEnvs := mgr.EnvVarMgr.EnvVarsByC[apicommonv1.UnprivilegedSingleAgentContainerName]
				assert.True(
					t,
					apiutils.IsEqualStruct(agentEnvs, exp.EnvVars),
					"Trace Agent ENVs \ndiff = %s", cmp.Diff(agentEnvs, exp.EnvVars),
				)
			}

			agentPorts := mgr.PortMgr.PortsByC[apicommonv1.UnprivilegedSingleAgentContainerName]
			assert.True(
				t,
				apiutils.IsEqualStruct(agentPorts, exp.Ports),
				"Core Agent Ports \ndiff = %s", cmp.Diff(agentPorts, exp.Ports),
			)
		},
	)
}
