package service

import (
	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"
)

// ---------- 在线用户：对应 SysUserOnlineServiceImpl.java ----------

// SelectOnlineList 在线用户列表，等价 SysUserOnlineController.list(L43-70)。
//
// 数据源是 Redis 里所有 login_tokens:<uuid> 键（L45 的 redisCache.keys(prefix + "*")），
// **不查数据库**。过滤规则严格照抄 L50-65 的四分支：
//
//   - 同时给了 ipaddr 与 userName：两者都相等才收（否则塞 null），见 selectOnlineByInfo(L60-67)
//   - 只给 ipaddr：selectOnlineByIpaddr(L26-32)
//   - 只给 userName：selectOnlineByUserName(L43-49)，且要求 user 非 null
//   - 都没给：loginUserToUserOnline(L76-94)
//
// 最后两步同样照抄：Collections.reverse（L67，按登录时间倒序）
// 与 removeAll(singleton(null))（L68，清掉过滤产生的 null 项）。
//
// DIFF(review): Java 的 getDataTable 里 total 是 new PageInfo(list).getTotal()，
// 因为没有 startPage()，所以 total 恒等于列表长度 —— Go 侧直接返回 len(rows)。
func SelectOnlineList(ipaddr, userName string) ([]entity.SysUserOnline, error) {
	keys, err := redisx.Keys(config.LoginTokenKey + "*")
	if err != nil {
		return nil, err
	}
	list := make([]entity.SysUserOnline, 0, len(keys))
	for _, key := range keys {
		var lu entity.LoginUser
		ok, err := redisx.GetObject(key, &lu)
		if err != nil || !ok {
			// 登录态已过期/反序列化失败：跳过，与 Java 的 getCacheObject 返回 null 后
			// loginUserToUserOnline 返回 null 再被 removeAll 清掉的效果一致。
			continue
		}
		var item *entity.SysUserOnline
		switch {
		case ipaddr != "" && userName != "":
			if lu.Ipaddr == ipaddr && lu.Username() == userName {
				item = loginUserToUserOnline(&lu)
			}
		case ipaddr != "":
			if lu.Ipaddr == ipaddr {
				item = loginUserToUserOnline(&lu)
			}
		case userName != "":
			if lu.User != nil && lu.Username() == userName {
				item = loginUserToUserOnline(&lu)
			}
		default:
			item = loginUserToUserOnline(&lu)
		}
		if item != nil {
			list = append(list, *item)
		}
	}
	// Collections.reverse(L67)
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
}

// loginUserToUserOnline 等价 SysUserOnlineServiceImpl.loginUserToUserOnline(L76-94)。
//
// user 或 user.user 为空时返回 nil（L78-81），调用方会把它过滤掉。
func loginUserToUserOnline(lu *entity.LoginUser) *entity.SysUserOnline {
	if lu == nil || lu.User == nil {
		return nil
	}
	online := &entity.SysUserOnline{
		TokenID:       lu.Token,
		UserName:      lu.Username(),
		Ipaddr:        lu.Ipaddr,
		LoginLocation: lu.LoginLocation,
		Browser:       lu.Browser,
		Os:            lu.Os,
		LoginTime:     lu.LoginTime,
	}
	if lu.User.Dept != nil {
		online.DeptName = lu.User.Dept.DeptName
	}
	return online
}

// ForceLogout 强退用户，等价 SysUserOnlineController.forceLogout(L78-82)：
// 直接删掉 login_tokens:<tokenId>，下一次请求会 401。
func ForceLogout(tokenID string) error {
	if tokenID == "" {
		return nil
	}
	return redisx.Del(config.LoginTokenKey + tokenID)
}
