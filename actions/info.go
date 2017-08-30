package actions

func Info() map[string]string {
	info := make(map[string]string)
	info["api_version"] = "2.0"
	return info
}
