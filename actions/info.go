package actions

func Info() map[string]string {
	info := make(map[string]string)
	info["api_version"] = "1.0"
	return info
}
