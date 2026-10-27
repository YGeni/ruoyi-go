// Package uax 解析 User-Agent 与 IP 归属地。
//
// 等价 ruoyi-common 的两个工具类：
//   - UserAgentUtils.getBrowser / getOperatingSystem（浏览器与操作系统）
//   - AddressUtils.getRealAddressByIP（登录归属地）
//   - IpUtils.internalIp（内网判定）
//
// 供 TokenService.setUserAgent（TokenService.java:163-171）在登录时写入 LoginUser，
// 最终显示在"系统监控 > 在线用户"页面。
package uax

import (
	"net"
	"regexp"
	"strings"
)

// UNKNOWN 未知地址，等价 AddressUtils.UNKNOWN（AddressUtils.java:25）。
const UNKNOWN = "XX XX"

// InternalIPLabel 内网地址文案，等价 AddressUtils 的内网分支（AddressUtils.java:31-34）。
const InternalIPLabel = "内网IP"

// UnknownName UserAgentUtils 对无法识别的 UA 返回的名字。
const UnknownName = "Unknown"

// Location 等价 AddressUtils.getRealAddressByIP（AddressUtils.java:27-52）。
//
// **不做外网查询**：application.yml:12 的 ruoyi.addressEnabled 是 false，
// Java 在关闭时直接返回 UNKNOWN，只有内网 IP 返回"内网IP"。
// 这里完全对齐该行为（不引入 whois.pconline 的外部依赖，也就不会有网络抖动拖慢登录）。
func Location(ip string) string {
	if InternalIP(ip) {
		return InternalIPLabel
	}
	return UNKNOWN
}

// InternalIP 等价 IpUtils.internalIp（IpUtils.java:77-81 + 私网三段判定）。
//
// 非 IPv4（例如 ::1 或主机名）一律视为内网：
// Java 的 textToNumericFormatV4 对 IPv6 返回 null，这里避免空指针并给出更合理的结果。
func InternalIP(ip string) bool {
	addr := net.ParseIP(ip)
	if addr == nil {
		return true
	}
	v4 := addr.To4()
	if v4 == nil {
		return true // IPv6（含 ::1）
	}
	if v4[0] == 0x0A { // 10.0.0.0/8
		return true
	}
	if v4[0] == 0xAC && v4[1] >= 0x10 && v4[1] <= 0x1F { // 172.16.0.0/12
		return true
	}
	if v4[0] == 0xC0 && v4[1] == 0xA8 { // 192.168.0.0/16
		return true
	}
	return ip == "127.0.0.1"
}

// 版本号抽取：取 UA 片段里的主版本号，如 "Chrome/141.0.7390.54" -> 141。
var versionRe = regexp.MustCompile(`/(\d+)(?:\.|$)`)

// Browser 等价 UserAgentUtils.getBrowser(userAgent).getName()，
// 返回形如 "Chrome 141" / "Edge 141" / "Firefox 115" / "Safari 17"。
//
// 判定顺序**必须**先 Edge 再 Chrome：Edge 的 UA 里同时含 "Edg/" 与 "Chrome/"。
func Browser(ua string) string {
	switch {
	case contains(ua, "Edg/"), contains(ua, "Edge/"):
		return withVersion(ua, "Edge", "Edg/", "Edge/")
	case contains(ua, "OPR/"), contains(ua, "Opera"):
		return withVersion(ua, "Opera", "OPR/", "Opera/")
	case contains(ua, "Firefox/"):
		return withVersion(ua, "Firefox", "Firefox/")
	case contains(ua, "Chrome/"):
		return withVersion(ua, "Chrome", "Chrome/")
	case contains(ua, "Safari/"):
		return withVersion(ua, "Safari", "Version/", "Safari/")
	}
	return UnknownName
}

// OS 等价 UserAgentUtils.getOperatingSystem(userAgent).getName()，
// 返回形如 "Windows 10" / "Mac OS X" / "Linux" / "Android" / "iOS"。
func OS(ua string) string {
	switch {
	case contains(ua, "Windows NT 10.0"):
		return "Windows 10"
	case contains(ua, "Windows NT 6.3"):
		return "Windows 8.1"
	case contains(ua, "Windows NT 6.2"):
		return "Windows 8"
	case contains(ua, "Windows NT 6.1"):
		return "Windows 7"
	case contains(ua, "Windows NT 6.0"):
		return "Windows Vista"
	case contains(ua, "Windows NT 5.1"), contains(ua, "Windows NT 5.2"):
		return "Windows XP"
	case contains(ua, "Windows"):
		return "Windows"
	case contains(ua, "Android"):
		return "Android"
	case contains(ua, "iPhone"), contains(ua, "iPad"), contains(ua, "iPod"):
		return "iOS"
	case contains(ua, "Mac OS X"), contains(ua, "Macintosh"):
		return "Mac OS X"
	case contains(ua, "Linux"):
		return "Linux"
	}
	return UnknownName
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// withVersion 在名字后拼主版本号；抽不到就只返回名字。
func withVersion(ua, name string, segments ...string) string {
	for _, seg := range segments {
		if i := strings.Index(ua, seg); i >= 0 {
			if m := versionRe.FindStringSubmatch(ua[i+len(seg)-1:]); m != nil {
				return name + " " + m[1]
			}
			return name
		}
	}
	return name
}
