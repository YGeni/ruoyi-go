package uax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	// 内网一律"内网IP"（AddressUtils.java:30-34）
	for _, ip := range []string{"127.0.0.1", "192.168.1.10", "10.0.0.7", "172.16.0.1", "172.31.255.255", "::1"} {
		assert.Equal(t, InternalIPLabel, Location(ip), ip)
	}
	// addressEnabled=false（application.yml:12）时公网不做查询，回落 UNKNOWN
	assert.Equal(t, UNKNOWN, Location("8.8.8.8"))
	assert.Equal(t, UNKNOWN, Location("223.5.5.5"))
	// 非 IP 输入不能 panic
	assert.Equal(t, InternalIPLabel, Location(""))
}

func TestInternalIP(t *testing.T) {
	assert.True(t, InternalIP("127.0.0.1"))
	assert.True(t, InternalIP("10.255.255.255"))
	assert.True(t, InternalIP("172.16.0.0"))
	assert.True(t, InternalIP("172.31.0.0"))
	assert.True(t, InternalIP("192.168.0.1"))
	// 边界：172.15 / 172.32 不属于私网
	assert.False(t, InternalIP("172.15.0.1"))
	assert.False(t, InternalIP("172.32.0.1"))
	assert.False(t, InternalIP("8.8.8.8"))
}

// TestBrowserEdgeBeforeChrome Edge 的 UA 同时含 Edg/ 与 Chrome/，
// 判定顺序反了会把它认成 Chrome —— 在线用户的"浏览器"列就会全错。
func TestBrowserEdgeBeforeChrome(t *testing.T) {
	edge := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"
	assert.Equal(t, "Edge 141", Browser(edge))

	chrome := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.7390.54 Safari/537.36"
	assert.Equal(t, "Chrome 141", Browser(chrome))

	ff := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:115.0) Gecko/20100101 Firefox/115.0"
	assert.Equal(t, "Firefox 115", Browser(ff))

	// Safari 的版本号在 Version/ 上，不是 Safari/
	safari := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
	assert.Equal(t, "Safari 17", Browser(safari))

	opera := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0"
	assert.Equal(t, "Opera 106", Browser(opera))

	assert.Equal(t, UnknownName, Browser("curl/8.4.0"))
	assert.Equal(t, UnknownName, Browser(""))
}

func TestOS(t *testing.T) {
	cases := map[string]string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)":              "Windows 10",
		"Mozilla/5.0 (Windows NT 6.1; WOW64)":                    "Windows 7",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)":        "Mac OS X",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8)":               "Android",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)": "iOS",
		"Mozilla/5.0 (X11; Linux x86_64)":                        "Linux",
		"curl/8.4.0":                                             UnknownName,
	}
	for ua, want := range cases {
		assert.Equal(t, want, OS(ua), ua)
	}
}
