package repository

import (
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
)

// nowTime 当前时间，用于显式写 update_time。
//
// 用 map 更新时 GORM 不触发 BeforeUpdate hook（见 user_repo.go 的同名说明），
// 必须自己写 update_time，否则与 Java 侧 update_time = sysdate() 的行为不一致。
func nowTime() time.Time { return time.Now() }

// SelectRoleNamesByUserName 查用户名所属角色名列表，
// 等价 SysRoleMapper.xml:81-84 的 selectRolesByUserName（selectRoleVo + u.user_name = ?）。
//
// 供 SysUserServiceImpl.selectUserRoleGroup(L138-147) 拼 roleGroup 使用：
// 空列表 -> 空串；否则用 "," 连接（Java 的 Collectors.joining(",")）。
//
// 注意 selectRoleVo 里没有 ORDER BY，Java 侧的拼接顺序即数据库返回顺序；
// Go 侧同样不加排序，避免与 Java 行为漂移。
func SelectRoleNamesByUserName(userName string) ([]string, error) {
	var names []string
	err := db.DB.Table("sys_role r").
		Select("DISTINCT r.role_name").
		Joins("LEFT JOIN sys_user_role ur ON ur.role_id = r.role_id").
		Joins("LEFT JOIN sys_user u ON u.user_id = ur.user_id").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Where("r.del_flag = ?", model.DelFlagExists).
		Where("u.user_name = ?", userName).
		Pluck("r.role_name", &names).Error
	return names, err
}

// SelectPostNamesByUserName 查用户名所属岗位名列表，
// 等价 SysPostMapper.xml:57-63 的 selectPostsByUserName。
//
// 与角色不同：sys_post 没有 del_flag，Java 的原 SQL 也不加任何过滤。
func SelectPostNamesByUserName(userName string) ([]string, error) {
	var names []string
	err := db.DB.Table("sys_post p").
		Select("p.post_name").
		Joins("LEFT JOIN sys_user_post up ON up.post_id = p.post_id").
		Joins("LEFT JOIN sys_user u ON u.user_id = up.user_id").
		Where("u.user_name = ?", userName).
		Pluck("p.post_name", &names).Error
	return names, err
}

// UpdateUserProfile 修改个人资料，等价 SysUserMapper.xml updateUser 的 profile 用法
// （SysUserServiceImpl.updateUserProfile(L340-343) 直接复用 updateUser）。
//
// Java 的 updateUser 是动态列，只更新非空的四个字段；
// Go 侧同样只写这四列 —— 这一点很关键：profile 接口绝不能让前端顺带改掉
// status / del_flag / dept_id 等敏感列。
func UpdateUserProfile(u *entity.SysUser) error {
	return db.DB.Table("sys_user").Where("user_id = ?", u.UserID).
		Updates(map[string]any{
			"nick_name":   u.NickName,
			"phonenumber": u.Phonenumber,
			"email":       u.Email,
			"sex":         u.Sex,
			"update_by":   u.UpdateBy,
			"update_time": nowTime(),
		}).Error
}

// JoinNames 把名字列表用 "," 连接；空列表返回空串。
//
// 等价 SysUserServiceImpl 里的 Collectors.joining(",")。
func JoinNames(names []string) string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n != "" {
			out = append(out, n)
		}
	}
	return strings.Join(out, ",")
}
