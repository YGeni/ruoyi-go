package service

import (
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOnlineUser 造一个"已登录"的 Redis 快照并写进 login_tokens:<uuid>。
//
// 与 newLoginUser 的区别：这里必须**真的写进 Redis**，
// 因为 SelectOnlineList 的数据源就是这些键（不查数据库）。
func newOnlineUser(t *testing.T, userID int64, ipaddr, browser, os string) *entity.LoginUser {
	t.Helper()
	ensureRedis(t)
	u, err := repository.SelectUserByID(userID)
	require.NoError(t, err)
	require.NotNil(t, u)
	lu := &entity.LoginUser{
		UserID:      u.UserID,
		DeptID:      u.DeptID,
		Token:       uuid.NewString(),
		User:        u,
		Ipaddr:      ipaddr,
		Browser:     browser,
		Os:          os,
		Permissions: []string{"*:*:*"},
	}
	require.NoError(t, RefreshToken(lu))
	t.Cleanup(func() { _ = redisx.Del(config.LoginTokenKey + lu.Token) })
	return lu
}

func ensureRedis(t *testing.T) {
	t.Helper()
	if redisx.Client() == nil {
		require.NoError(t, redisx.Init(config.C.Redis))
	}
}

// 只统计本用例造出来的登录态，避免并发/历史键干扰断言。
func countMine(t *testing.T, mine map[string]bool, ipaddr, userName string) int {
	t.Helper()
	rows, err := SelectOnlineList(ipaddr, userName)
	require.NoError(t, err)
	n := 0
	for _, r := range rows {
		if mine[r.TokenID] {
			n++
		}
	}
	return n
}

// TestSelectOnlineList 等价 SysUserOnlineController.list(L43-70) 的四分支过滤。
func TestSelectOnlineList(t *testing.T) {
	ensureRedis(t)
	a := newOnlineUser(t, uAdmin, "10.0.0.1", "Chrome 141", "Windows 10")
	b := newOnlineUser(t, uAdmin, "10.0.0.2", "Safari 17", "Mac OS X")
	mine := map[string]bool{a.Token: true, b.Token: true}

	// 无过滤：两个都收
	rows, err := SelectOnlineList("", "")
	require.NoError(t, err)
	assert.Equal(t, 2, countMine(t, mine, "", ""))

	// 字段映射：tokenId / userName / ipaddr / browser / os 来自 Redis 快照，
	// deptName 来自 LoginUser.User.Dept（SysUserOnlineServiceImpl.java:90-93）。
	byToken := map[string]entity.SysUserOnline{}
	for _, r := range rows {
		byToken[r.TokenID] = r
	}
	ra, ok := byToken[a.Token]
	require.True(t, ok, "本用例造的登录态必须出现在列表里")
	assert.Equal(t, "admin", ra.UserName)
	assert.Equal(t, "10.0.0.1", ra.Ipaddr)
	assert.Equal(t, "Chrome 141", ra.Browser)
	assert.Equal(t, "Windows 10", ra.Os)
	assert.NotZero(t, ra.LoginTime, "loginTime 由 RefreshToken 写入")
	if a.User != nil && a.User.Dept != nil {
		assert.Equal(t, a.User.Dept.DeptName, ra.DeptName)
	}

	// 只给 ipaddr
	assert.Equal(t, 1, countMine(t, mine, "10.0.0.1", ""))
	assert.Equal(t, 0, countMine(t, mine, "9.9.9.9", ""))
	// 只给 userName
	assert.Equal(t, 2, countMine(t, mine, "", "admin"))
	assert.Equal(t, 0, countMine(t, mine, "", "nobody"))
	// 同时给：两者都命中才收（selectOnlineByInfo L60-67）
	assert.Equal(t, 1, countMine(t, mine, "10.0.0.2", "admin"))
	assert.Equal(t, 0, countMine(t, mine, "10.0.0.2", "nobody"))
}

// TestForceLogout 等价 SysUserOnlineController.forceLogout(L78-82)。
func TestForceLogout(t *testing.T) {
	ensureRedis(t)
	lu := newOnlineUser(t, uAdmin, "10.0.0.9", "Firefox 115", "Linux")
	mine := map[string]bool{lu.Token: true}
	require.Equal(t, 1, countMine(t, mine, "", ""))

	require.NoError(t, ForceLogout(lu.Token))
	assert.Equal(t, 0, countMine(t, mine, "", ""), "强退后必须查不到")

	// 幂等：强退不存在的 token 不报错（Java 的 deleteObject 同理）
	assert.NoError(t, ForceLogout(lu.Token))
	assert.NoError(t, ForceLogout(""))
}

// TestLoginUserToUserOnlineSkipNilUser 快照里 user 为空时必须返回 nil，
// 由 SelectOnlineList 过滤掉（SysUserOnlineServiceImpl.java:78-81 的 null 分支 +
// Controller L68 的 removeAll(singleton(null))）。
func TestLoginUserToUserOnlineSkipNilUser(t *testing.T) {
	assert.Nil(t, loginUserToUserOnline(nil))
	assert.Nil(t, loginUserToUserOnline(&entity.LoginUser{}))
	lu := &entity.LoginUser{User: &entity.SysUser{UserName: "admin"}}
	assert.NotNil(t, loginUserToUserOnline(lu))
	assert.Equal(t, "admin", loginUserToUserOnline(lu).UserName)
}
