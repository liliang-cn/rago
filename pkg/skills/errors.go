package skills

import "fmt"

// Error definitions
var (
	ErrSkillNotFound      = fmt.Errorf("skill not found")
	ErrSkillDisabled      = fmt.Errorf("skill is disabled")
	ErrVariableRequired   = fmt.Errorf("required variable missing")
	ErrVariableInvalid    = fmt.Errorf("variable value is invalid")
	ErrExecutionFailed    = fmt.Errorf("skill execution failed")
	ErrCommandNotAllowed  = fmt.Errorf("command injection not allowed")
	ErrStepNotFound       = fmt.Errorf("step not found")
	ErrConfirmationDenied = fmt.Errorf("confirmation denied")
)

// SkillError represents an error with additional context
type SkillError struct {
	Code    string
	Message string
	SkillID string
	Err     error
}

// Error returns the error message
func (e *SkillError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *SkillError) Unwrap() error {
	return e.Err
}

// NewSkillError creates a new skill error
func NewSkillError(code, message, skillID string, err error) *SkillError {
	return &SkillError{
		Code:    code,
		Message: message,
		SkillID: skillID,
		Err:     err,
	}
}
