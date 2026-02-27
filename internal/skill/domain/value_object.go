package domain

type SkillMetadata struct {
	Category    string   `yaml:"category"`
	Complexity  string   `yaml:"complexity"`
	Permissions []string `yaml:"permissions"`
}

func (sm *SkillMetadata) IsValid() bool {
	if sm == nil {
		return true
	}

	if sm.Complexity != "" && !isValidComplexity(sm.Complexity) {
		return false
	}

	return true
}

func isValidComplexity(complexity string) bool {
	validLevels := map[string]bool{
		"simple":  true,
		"medium":  true,
		"complex": true,
	}
	return validLevels[complexity]
}

type SkillValidationError struct {
	Field   string
	Message string
	Code    SkillValidationCode
}

func (e *SkillValidationError) Error() string {
	if e.Code != "" {
		return string(e.Code) + ": " + e.Field + " - " + e.Message
	}
	return e.Field + " - " + e.Message
}

type SkillValidationCode string

const (
	CodeMissingField  SkillValidationCode = "MISSING_FIELD"
	CodeInvalidFormat SkillValidationCode = "INVALID_FORMAT"
	CodeInvalidValue  SkillValidationCode = "INVALID_VALUE"
	CodeInvalidLength SkillValidationCode = "INVALID_LENGTH"
)

type SkillExistsError struct {
	ID SkillID
}

func (e *SkillExistsError) Error() string {
	return "skill with ID " + string(e.ID) + " already exists"
}

type SkillNotFoundError struct {
	ID SkillID
}

func (e *SkillNotFoundError) Error() string {
	return "skill with ID " + string(e.ID) + " not found"
}
