package utils

type ValidationError struct {
	Field   string `json:"field"`   // ชื่อฟิลด์ที่มีปัญหา
	Message string `json:"message"` // ข้อความอธิบายที่เข้าใจง่าย
}

func MsgForTag(tag string, param string) string {
	switch tag {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Should be at least " + param + " characters"
	case "max":
		return "Should not exceed " + param + " characters"
	case "oneof":
		return "Must be one of the following: " + param
	}
	return "Invalid value"
}
