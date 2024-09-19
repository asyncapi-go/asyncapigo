package parser

func mapType(source string) (name string) {
	switch source {
	case "int8", "uint8", "byte", "int16", "uint16", "int32", "rune", "uint32", "int64", "uint64", "int", "uint",
		"uintptr":
		return "integer"
	case "float32", "float64":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	default:
		return ""
	}
}
