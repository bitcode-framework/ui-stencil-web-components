package plugin

import "encoding/json"

const (
	MsgTypeExecute         = "execute"
	MsgTypeBridgeResponse  = "bridge_response"
	MsgTypeBridgeRequest   = "bridge_request"
	MsgTypeExecuteComplete = "execute_complete"
	MsgTypeExecuteError    = "execute_error"
)

// Message is the bidirectional JSON-RPC envelope between Go and child processes (Node.js, Python).
type Message struct {
	Type   string          `json:"type"`
	ID     int             `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *MessageError   `json:"error,omitempty"`
	TxID   string          `json:"txId,omitempty"`
}

// MessageError carries structured error information through JSON-RPC.
type MessageError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Stack   string         `json:"stack,omitempty"`
	Retryable bool         `json:"retryable,omitempty"`
}

// ExecuteParams is sent from Go to Node.js to request script execution.
type ExecuteParams struct {
	Script        string         `json:"script"`
	Params        map[string]any `json:"params"`
	Module        string         `json:"module"`
	Session       SessionInfo    `json:"session"`
	SecurityRules map[string]any `json:"securityRules"`
}

// SessionInfo is the session data injected into the script sandbox.
type SessionInfo struct {
	UserID   string         `json:"userId"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	TenantID string         `json:"tenantId"`
	Groups   []string       `json:"groups"`
	Locale   string         `json:"locale"`
	Context  map[string]any `json:"context"`
}

// BridgeRequest is parsed from a bridge_request message sent by Node.js.
type BridgeRequest struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
	TxID   string         `json:"txId,omitempty"`
}


