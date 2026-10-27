package entity

// SysUserOnline 在线用户，等价于 ruoyi-system/.../system/domain/SysUserOnline.java。
//
// **不是数据库表**：数据全部来自 Redis 里 login_tokens:<uuid> 的 LoginUser 快照，
// 由 SysUserOnlineServiceImpl.loginUserToUserOnline(L76-94) 组装。
//
// 字段与 JSON 必须满足前端 src/views/monitor/online/index.vue:37-43 的列绑定
// （tokenId / userName / deptName / ipaddr / loginLocation / browser / os / loginTime）。
type SysUserOnline struct {
	TokenID       string `json:"tokenId"`
	DeptName      string `json:"deptName"`
	UserName      string `json:"userName"`
	Ipaddr        string `json:"ipaddr"`
	LoginLocation string `json:"loginLocation"`
	Browser       string `json:"browser"`
	Os            string `json:"os"`
	LoginTime     int64  `json:"loginTime"` // 毫秒时间戳
}
