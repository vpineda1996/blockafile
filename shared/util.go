package shared

func TruncateString(str string, n int) string {
	if len(str) > n {
		return str[0:n]
	}
	return str
}
