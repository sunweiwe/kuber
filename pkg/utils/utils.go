package utils

func StrOrDef(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}
