package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
)

func TestExtensionController_BuildContainerAndVolumes(t *testing.T) {
	assertion := assert.New(t)

	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
		"", "", "test-feature", "", "",
		nil, nil, nil,
	)

	t.Run("happy path", func(t *testing.T) {
		eec := NewExtensionController(stsProperties)
		container := eec.BuildContainer()

		assertion.NotEmpty(container.ReadinessProbe, "Expected readiness probe is defined")
		assertion.Equal("/readyz", container.ReadinessProbe.HTTPGet.Path, "Expected there is a readiness probe at /readyz")
		assertion.Empty(container.LivenessProbe, "Expected there is no liveness probe (not implemented)")
		assertion.Empty(container.StartupProbe, "Expected there is no startup probe")

		for _, port := range []int32{eecIngestPort} {
			assertion.Truef(kubeobjects.PortIsIn(container.Ports, port), "Expected that EEC container defines port %d", port)
		}

		for _, mountPath := range []string{
			activeGateConfigDir,
			dataSourceStartupArgsMountPoint,
			dataSourceAuthTokenMountPoint,
			statsdMetadataMountPoint,
			extensionsLogsDir,
			statsdLogsDir,
		} {
			assertion.Truef(kubeobjects.MountPathIsIn(container.VolumeMounts, mountPath), "Expected that EEC container defines mount point %s", mountPath)
		}

		for _, envVar := range []string{
			"TenantId", "ServerUrl", "EecIngestPort",
		} {
			assertion.Truef(kubeobjects.EnvVarIsIn(container.Env, envVar), "Expected that EEC container defined environment variable %s", envVar)
		}
	})

	t.Run("volumes vs volume mounts", func(t *testing.T) {
		eec := NewExtensionController(stsProperties)
		statsd := NewStatsd(stsProperties)
		volumes := buildVolumes(stsProperties, []kubeobjects.ContainerBuilder{eec, statsd})

		container := eec.BuildContainer()
		for _, volumeMount := range container.VolumeMounts {
			assertion.Truef(kubeobjects.VolumeIsDefined(volumes, volumeMount.Name), "Expected that volume mount %s has a predefined pod volume", volumeMount.Name)
		}
	})
}

func TestBuildEecConfigMapName(t *testing.T) {
	t.Run("happy case", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("dynakube", "activegate")
		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMapName)
	})

	t.Run("happy case, capitalized and with spaces", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("DynaKube", "Active Gate")
		assert.Equal(t, "DynaKube-Active_Gate-eec-config", eecConfigMapName)
	})

	t.Run("empty module", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("DynaKube", "")
		assert.Equal(t, "", eecConfigMapName)
	})

	t.Run("whitespace-only module", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("DynaKube", " 		")
		assert.Equal(t, "DynaKube-___-eec-config", eecConfigMapName)
	})
}
