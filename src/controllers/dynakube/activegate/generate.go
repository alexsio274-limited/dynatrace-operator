package activegate

import "github.com/pkg/errors"

const (
	CommunicationEndpointsName = "communication-endpoints"
	TenantTokenName            = "tenant-token"
	TenantUuidName             = "tenant-uuid"
)

func (r *Reconciler) getActiveGateTenantInfo() (map[string][]byte, error) {
	tenantInfo, err := r.dtc.GetActiveGateTenantInfo()

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		TenantUuidName:             []byte(tenantInfo.UUID),
		TenantTokenName:            []byte(tenantInfo.Token),
		CommunicationEndpointsName: []byte(tenantInfo.Endpoints),
	}, nil
}
