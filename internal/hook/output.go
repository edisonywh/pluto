package hook

import "encoding/json"

type preToolUseOutput struct {
	HookSpecificOutput hookSpecific `json:"hookSpecificOutput"`
}

type hookSpecific struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

// AllowOutput returns the JSON string that tells Claude Code to allow the action.
func AllowOutput() string {
	b, _ := json.Marshal(preToolUseOutput{HookSpecificOutput: hookSpecific{
		HookEventName:      "PreToolUse",
		PermissionDecision: "allow",
	}})
	return string(b)
}

// DenyOutput returns the JSON string that tells Claude Code to deny the action with a reason.
func DenyOutput(message string) string {
	b, _ := json.Marshal(preToolUseOutput{HookSpecificOutput: hookSpecific{
		HookEventName:            "PreToolUse",
		PermissionDecision:       "deny",
		PermissionDecisionReason: message,
	}})
	return string(b)
}
