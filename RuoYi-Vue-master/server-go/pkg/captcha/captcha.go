// Package captcha 复刻 CaptchaController 的验证码生成与校验。
//
// 对应实现：
// -ruoyi-
//
//	Redis key = captcha_codes:<uuid>（L57-58），TTL = CAPTCHA_EXPIRATION = 2 分钟（L78），
//	math 型生成串形如 "3+5=?@8"，@ 前为展示串、@ 后为答案（L67-69），
//	返回体字段 captchaEnabled / uuid / img（L90-91），img 为裸 base64（不含 data URI 前缀）。
//
// -ruoyi-
package captcha

import (
	"image/color"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"github.com/mojocn/base64Captcha"
)

const dataURIPrefixMark = "base64,"

// redisStore 把验证码答案存进 Redis，替代 base64Captcha 的内存 Store，
// 以便与 Java 端共用同一份 captcha_codes: 缓存。
type redisStore struct {
	prefix string
	ttl    time.Duration
}

func (s *redisStore) Set(id string, value string) error {
	return redisx.Set(s.prefix+id, value, s.ttl)
}

func (s *redisStore) Get(id string, clear bool) string {
	v, ok, err := redisx.Get(s.prefix + id)
	if err != nil || !ok {
		return ""
	}
	if clear {
		_ = redisx.Del(s.prefix + id)
	}
	return v
}

func (s *redisStore) Verify(id, answer string, clear bool) bool {
	return strings.EqualFold(s.Get(id, clear), strings.TrimSpace(answer))
}

func newStore() *redisStore {
	return &redisStore{
		prefix: config.CaptchaCodeKey,
		ttl:    time.Duration(config.CaptchaExpiration) * time.Minute,
	}
}

// driver 按 ruoyi.captchaType 选择 math / char。
//
// base64Captcha v1.3.8 的构造器需要显式传入 fontsStorage 与 fonts，
// 传 nil 时库内部会回退到内置字体（driver_math.go:39-51）。
func driver() base64Captcha.Driver {
	white := &color.RGBA{R: 255, G: 255, B: 255, A: 255}
	// Java 画布为 160x60
	const height, width = 60, 160
	if config.C.RuoYi.CaptchaType == "char" {
		// char 型对应 4 位字符验证码长度 4）
		return base64Captcha.NewDriverString(height, width, 0, 0, 4,
			base64Captcha.TxtNumbers+base64Captcha.TxtAlphabet, white, nil, nil)
	}
	return base64Captcha.NewDriverMath(height, width, 0, 0, white, nil, nil)
}

// Generate 生成验证码，返回 uuid 与**裸 base64**（前端自行拼接 data URI 前缀）。
//
// DIFF(review): Java 用 ImageIO.write(image, "jpg")，输出 JPEG；Go 的 base64Captcha
// 驱动输出 PNG。前端 login.vue:143-149 固定拼接 "data:image/gif;base64,"，
// 原实现同样是"后缀与实际格式不一致"，浏览器按内容嗅探渲染，行为一致。
// DIFF(review): Java 的 uuid 用 IdUtils.simpleUUID()（32 位无横杠），Go 侧沿用
// base64Captcha 生成的随机 id，仅作为 Redis key 后缀，前端不解析其格式。
func Generate() (uuid string, img string, err error) {
	c := base64Captcha.NewCaptcha(driver(), newStore())
	id, b64s, _, err := c.Generate()
	if err != nil {
		return "", "", err
	}
	return id, stripDataURI(b64s), nil
}

// Verify 校验验证码（大小写不敏感，校验后即删除，与 Java L122-127 一致）。
func Verify(uuid, code string) bool {
	return newStore().Verify(uuid, code, true)
}

func stripDataURI(b64s string) string {
	if i := strings.Index(b64s, dataURIPrefixMark); i >= 0 {
		return b64s[i+len(dataURIPrefixMark):]
	}
	return b64s
}
