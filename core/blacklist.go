package core

func BlacklistCheck(domain string, blacklist []string) bool {
	// 检查域名是否在黑名单中
	for _, entry := range blacklist {
		if entry == domain || entry == "*."+domain {
			return true
		}
	}
	return false
}
