package skill

import "errors"

var (
	ErrSkillNotFound         = errors.New("skill not found")
	ErrInvalidFrontmatter    = errors.New("invalid frontmatter")
	ErrInvalidInvocation     = errors.New("invalid skill invocation")
	ErrSkillNotUserInvocable = errors.New("skill is not user invocable")
	ErrCommandTimeout        = errors.New("command execution timeout")
)
