package statsdingest

import "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"

const (
	EecContainerName       = consts.ActiveGateContainerName + "-eec"
	StatsDContainerName    = consts.ActiveGateContainerName + "-statsd"
	StatsDIngestPortName   = "statsd"
	StatsDIngestPort       = 18125
	StatsDIngestTargetPort = "statsd-port"
)
