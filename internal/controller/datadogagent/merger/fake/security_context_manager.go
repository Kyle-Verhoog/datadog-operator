package fake

import (
	"testing"

	commonv1 "github.com/DataDog/datadog-operator/api/datadoghq/common/v1"

	v1 "k8s.io/api/core/v1"
)

// SecurityContextManager is a mock type for the SecurityContextManager type
type SecurityContextManager struct {
	CapabilitiesByC map[commonv1.AgentContainerName][]v1.Capability

	t testing.TB
}

// AddCapabilitiesToContainer provides a mock function with given fields: capabilities, containerName
func (_m *SecurityContextManager) AddCapabilitiesToContainer(capabilities []v1.Capability, containerName commonv1.AgentContainerName) {
	_m.CapabilitiesByC[containerName] = append(_m.CapabilitiesByC[containerName], capabilities...)
}

// NewFakeSecurityContextManager creates a new instance of SecurityContextManager. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewFakeSecurityContextManager(t testing.TB) *SecurityContextManager {
	return &SecurityContextManager{
		CapabilitiesByC: make(map[commonv1.AgentContainerName][]v1.Capability),
		t:               t,
	}
}
