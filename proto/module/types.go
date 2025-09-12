// Simple types for Phase 1 - avoiding protobuf complexity
package module

// Action types
type Action int32

const (
	Action_ACTION_CONTINUE Action = 1
	Action_ACTION_BLOCK    Action = 2
)

func (x Action) String() string {
	switch x {
	case Action_ACTION_CONTINUE:
		return "ACTION_CONTINUE"
	case Action_ACTION_BLOCK:
		return "ACTION_BLOCK"
	default:
		return "ACTION_UNKNOWN"
	}
}

// Health status types
type HealthStatus int32

const (
	HealthStatus_HEALTH_STATUS_HEALTHY   HealthStatus = 1
	HealthStatus_HEALTH_STATUS_UNHEALTHY HealthStatus = 2
)

func (x HealthStatus) String() string {
	switch x {
	case HealthStatus_HEALTH_STATUS_HEALTHY:
		return "HEALTHY"
	case HealthStatus_HEALTH_STATUS_UNHEALTHY:
		return "UNHEALTHY"
	default:
		return "UNKNOWN"
	}
}

// Request/Response messages
type ProcessRequestRequest struct {
	RequestId string `json:"request_id,omitempty"`
	TenantId  string `json:"tenant_id,omitempty"`
	Provider  string `json:"provider,omitempty"`
}

type ProcessRequestResponse struct {
	Action           Action            `json:"action,omitempty"`
	ProcessingTimeMs int64             `json:"processing_time_ms,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type HealthRequest struct{}

type HealthResponse struct {
	Status  HealthStatus      `json:"status,omitempty"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Getters for compatibility
func (x *ProcessRequestRequest) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *ProcessRequestRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *ProcessRequestRequest) GetProvider() string {
	if x != nil {
		return x.Provider
	}
	return ""
}

func (x *ProcessRequestResponse) GetAction() Action {
	if x != nil {
		return x.Action
	}
	return Action_ACTION_CONTINUE
}

func (x *ProcessRequestResponse) GetProcessingTimeMs() int64 {
	if x != nil {
		return x.ProcessingTimeMs
	}
	return 0
}

func (x *ProcessRequestResponse) GetAnnotations() map[string]string {
	if x != nil {
		return x.Annotations
	}
	return nil
}

func (x *ProcessRequestResponse) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *HealthResponse) GetStatus() HealthStatus {
	if x != nil {
		return x.Status
	}
	return HealthStatus_HEALTH_STATUS_HEALTHY
}

func (x *HealthResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *HealthResponse) GetDetails() map[string]string {
	if x != nil {
		return x.Details
	}
	return nil
}
