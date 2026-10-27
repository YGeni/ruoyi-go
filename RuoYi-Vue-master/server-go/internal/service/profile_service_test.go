package service

import (
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件守的是用户契约里的四条（前端直接依赖，改错页面就白）：
//
//	1. GET  /system/user/profile 的 user / roleGroup / postGroup 三个键
//	2. POST /system/user/profile/avatar 同步 Redis + 删旧文件
//	3. PUT  /system/user/profile/updatePwd 更新 pwd_update_date
//	4. 头像走 IMAGE_EXTENSION 白名单且重命名
//
// 零副作用：DB 改动发生在 TestMain 的事务里（滚回），
// Redis 用一次性 uuid 键并在 defer 里删除，磁盘用 t.TempDir()。

// newLoginUser 造一个"已登录"的 LoginUser（Redis 键为一次性 uuid）。
func newLoginUser(t *testing.T, userID int64) *entity.LoginUser {
	t.Helper()
	// service 包的 TestMain 只初始化了 MySQL；profile 用例要验证
	// "Redis LoginUser 同步"，因此这里按需补一次 Redis 连接。
	if redisx.Client() == nil {
		require.NoError(t, redisx.Init(config.C.Redis))
	}
	u, err := repository.SelectUserByID(userID)
	require.NoError(t, err)
	require.NotNil(t, u)
	lu := &entity.LoginUser{
		UserID: u.UserID,
		DeptID: u.DeptID,
		Token:  uuid.NewString(),
		User:   u,
		Ipaddr: "127.0.0.1",
	}
	t.Cleanup(func() {
		_ = redisx.Del(config.LoginTokenKey + lu.Token)
	})
	return lu
}

// ---------- 契约 1：roleGroup / postGroup ----------

// TestSelectUserRoleGroupPostGroup 等价 SysUserServiceImpl:138-164 的两个 group 查询。
//
// 种子数据：admin(1) 挂 role 1「超级管理员」+ post 1「董事长」；
//
//	ry(2)   挂 role 2「普通角色」  + post 2「项目经理」。
func TestSelectUserRoleGroupPostGroup(t *testing.T) {
	cases := []struct {
		userName      string
		wantRoleGroup string
		wantPostGroup string
	}{
		{"admin", "超级管理员", "董事长"},
		{"ry", "普通角色", "项目经理"},
	}
	for _, tc := range cases {
		rg, err := SelectUserRoleGroup(tc.userName)
		require.NoError(t, err)
		pg, err := SelectUserPostGroup(tc.userName)
		require.NoError(t, err)
		t.Logf("%s -> roleGroup=%q postGroup=%q", tc.userName, rg, pg)
		assert.Equal(t, tc.wantRoleGroup, rg, "%s 的 roleGroup", tc.userName)
		assert.Equal(t, tc.wantPostGroup, pg, "%s 的 postGroup", tc.userName)
	}

	// 不存在的用户必须是空串而不是 null：Java 的 StringUtils.EMPTY，
	// 前端把空串当空值渲染，null 会显示 "null"。
	rg, err := SelectUserRoleGroup("__not_exist__")
	require.NoError(t, err)
	assert.Equal(t, "", rg)
	pg, err := SelectUserPostGroup("__not_exist__")
	require.NoError(t, err)
	assert.Equal(t, "", pg)
}

// ---------- 契约 3：修改资料 ----------

func TestUpdateProfile(t *testing.T) {
	lu := newLoginUser(t, uAdmin)
	origNick := lu.User.NickName
	t.Cleanup(func() {
		// 事务会滚回，这里只需保证不影响后续用例对同一 LoginUser 的断言
		lu.User.NickName = origNick
	})

	err := UpdateProfile(lu, &entity.SysUser{
		UserID:      lu.UserID,
		NickName:    "冒烟昵称",
		Phonenumber: "15888888888",
		Email:       "smoke@ruoyi.com",
		Sex:         "0",
	})
	require.NoError(t, err)

	// DB 已改（事务内可读）
	got, err := repository.SelectUserByID(uAdmin)
	require.NoError(t, err)
	assert.Equal(t, "冒烟昵称", got.NickName)
	assert.Equal(t, "15888888888", got.Phonenumber)
	assert.Equal(t, "smoke@ruoyi.com", got.Email)
	assert.Equal(t, "0", got.Sex)

	// Redis LoginUser 同步（不等退出重登）
	cached, ok, err := GetLoginUser(lu.Token)
	require.NoError(t, err)
	require.True(t, ok, "SysProfileController.java:82 的 setLoginUser 必须有")
	assert.Equal(t, "冒烟昵称", cached.User.NickName)

	// 敏感列不能被 profile 接口改掉
	assert.Equal(t, "0", got.Status)
	assert.Equal(t, "0", got.DelFlag)

	// 手机号重复 -> 报错（L71-74）
	lu2 := newLoginUser(t, uRy)
	err = UpdateProfile(lu2, &entity.SysUser{UserID: uRy, Phonenumber: "15888888888", Email: "", NickName: "", Sex: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "手机号码已存在")
}

// ---------- 契约 3：修改密码 ----------

func TestUpdatePwd(t *testing.T) {
	lu := newLoginUser(t, uAdmin)

	// 旧密码错误
	err := UpdatePwd(lu, "wrong-old", "newpass123")
	require.Error(t, err)
	assert.Equal(t, "修改密码失败，旧密码错误", err.Error())

	// 新密码与旧密码相同
	err = UpdatePwd(lu, "admin123", "admin123")
	require.Error(t, err)
	assert.Equal(t, "新密码不能与旧密码相同", err.Error())

	// 成功路径，同时覆盖关键回归：
	//
	// Redis 快照里**没有**密码（SysUser.password 是 json:"-"，
	// 对应 Java 的 @JsonProperty(WRITE_ONLY)），线上从 Redis 取回的 LoginUser
	// 其 User.Password 一定是空串。这里刻意清空后再调用，
	// 逼 UpdatePwd 必须回 DB 取散列 —— 实测踩到过：
	// 依赖 lu.User.Password 会让正确的旧密码也被判成"旧密码错误"。
	before, err := repository.SelectUserByID(uAdmin)
	require.NoError(t, err)
	oldHash := before.Password
	oldPwdDate := before.PwdUpdateDate

	newLU := newLoginUser(t, uAdmin)
	newLU.User.Password = ""
	require.NoError(t, UpdatePwd(newLU, "admin123", "brandnew456"),
		"必须从 DB 取密码，不能依赖 Redis 快照里的 lu.User.Password")

	after, err := repository.SelectUserByID(uAdmin)
	require.NoError(t, err)
	assert.NotEqual(t, oldHash, after.Password, "密码散列必须变")
	assert.True(t, securityx.Matches("brandnew456", after.Password))
	require.NotNil(t, after.PwdUpdateDate)
	if oldPwdDate != nil {
		assert.False(t, after.PwdUpdateDate.Before(*oldPwdDate),
			"pwd_update_date 必须被刷新，否则 isDefaultModifyPwd / isPasswordExpired 会漂移")
	}

	// Redis 里的 LoginUser 也要同步 pwd_update_date（L113-115）。
	//
	// 注意**不能**断言 cached.User.Password 等于新散列：Java 的 SysUser.password
	// 是 @JsonProperty(WRITE_ONLY)（SysUser.java:202），Go 侧对应 json:"-"，
	// 密码**不会**被序列化进 Redis 快照（否则就把散列明文写进缓存了）。
	// 真正影响前端的只有 pwd_update_date。
	cached, ok, err := GetLoginUser(newLU.Token)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Empty(t, cached.User.Password, "密码不得进入 Redis 快照")
	require.NotNil(t, cached.User.PwdUpdateDate)
	assert.False(t, cached.User.PwdUpdateDate.Before(*oldPwdDate), "缓存里的 pwd_update_date 必须刷新")
}

// ---------- 契约 2 / 4：头像上传 ----------

// pngBytes 最小合法 PNG（1x1 透明），只做落盘校验用，不需要真解码。
var pngBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
}

func TestUpdateAvatar(t *testing.T) {
	// 把上传根目录指到临时目录，绝不能污染仓库里的 ./uploadPath
	origProfile := config.C.RuoYi.Profile
	config.C.RuoYi.Profile = t.TempDir()
	t.Cleanup(func() { config.C.RuoYi.Profile = origProfile })

	lu := newLoginUser(t, uAdmin)
	// 造一个"旧头像"文件，验证替换时被删除（SysProfileController.java:134-138）
	oldAvatar := "/profile/avatar/old/old.png"
	oldAbs := filepath.Join(config.C.RuoYi.GetProfile(), filepath.FromSlash(strings.TrimPrefix(oldAvatar, "/profile")))
	require.NoError(t, os.MkdirAll(filepath.Dir(oldAbs), 0o755))
	require.NoError(t, os.WriteFile(oldAbs, pngBytes, 0o644))
	lu.User.Avatar = oldAvatar

	avatar, err := UpdateAvatar(lu, fileHeaderFor(t, "me.png", pngBytes))
	require.NoError(t, err)
	t.Logf("新头像: %s", avatar)

	// 契约：返回 /profile/avatar/... 且 UUID 重命名（不保留 me.png）
	assert.True(t, strings.HasPrefix(avatar, "/profile/avatar/"), avatar)
	assert.NotContains(t, avatar, "me.png", "头像必须重命名（Java 传 useCustomNaming=true）")
	assert.Len(t, strings.TrimSuffix(filepath.Base(avatar), ".png"), 36)

	// 副作用①：DB 写入
	got, err := repository.SelectUserByID(uAdmin)
	require.NoError(t, err)
	assert.Equal(t, avatar, got.Avatar)

	// 副作用②：旧头像文件被删除
	_, statErr := os.Stat(oldAbs)
	assert.True(t, os.IsNotExist(statErr), "旧头像文件必须被删除：%s", oldAbs)

	// 副作用③：Redis LoginUser 同步（不然不退出重登看不到新头像）
	cached, ok, err := GetLoginUser(lu.Token)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, avatar, cached.User.Avatar)

	// 新文件确实落盘
	newAbs := filepath.Join(config.C.RuoYi.GetProfile(), filepath.FromSlash(strings.TrimPrefix(avatar, "/profile")))
	b, err := os.ReadFile(newAbs)
	require.NoError(t, err)
	assert.Equal(t, pngBytes, b)
}

func TestUpdateAvatarRejectsNonImage(t *testing.T) {
	origProfile := config.C.RuoYi.Profile
	config.C.RuoYi.Profile = t.TempDir()
	t.Cleanup(func() { config.C.RuoYi.Profile = origProfile })

	lu := newLoginUser(t, uAdmin)
	_, err := UpdateAvatar(lu, fileHeaderFor(t, "evil.exe", []byte("MZ")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evil.exe")
	assert.Contains(t, err.Error(), "bmp")
}

// fileHeaderFor 用本地字节构造真实的 multipart.FileHeader（带可 Open 的内容）。
func fileHeaderFor(t *testing.T, name string, content []byte) *multipart.FileHeader {
	t.Helper()
	src := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(src, content, 0o644))

	var sb strings.Builder
	w := multipart.NewWriter(&sb)
	part, err := w.CreateFormFile("avatarfile", name)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	form, err := multipart.NewReader(strings.NewReader(sb.String()), w.Boundary()).ReadForm(10 << 20)
	require.NoError(t, err)
	return form.File["avatarfile"][0]
}
