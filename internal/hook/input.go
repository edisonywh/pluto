package hook

// PermissionRequest is the JSON payload Claude Code sends to a PermissionRequest hook.
type PermissionRequest struct {
	SessionID     string    `json:"session_id"`
	HookEventName string    `json:"hook_event_name"`
	ToolName      string    `json:"tool_name"`
	ToolInput     ToolInput `json:"tool_input"`
}

// ToolInput holds the tool-specific fields for ExitPlanMode.
type ToolInput struct {
	Plan string `json:"plan"`
}
