// Package entity 定义与 31 张表一一对应的 GORM 模型。
//
// 命名与列映射严格对齐 sys_user 的实际列（无 login_name，登录名就是 user_name）：
// user_id dept_id user_name nick_name user_type email phonenumber sex avatar password
// status del_flag login_ip login_date pwd_update_date create_by create_time
// update_by update_time remark
package entity

import (
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model"

	"gorm.io/gorm"
)

// SysUser 用户信息表 sys_user。
//
// 关联关系对应 SysUserMapper.xml:27-28 的 resultMap：
//
//	<association property="dept"  .../>  -> Preload("Dept")
//	<collection  property="roles" .../>  -> Preload("Roles")（many2many: sys_user_role）
type SysUser struct {
	model.BaseModel
	UserID        int64      `gorm:"column:user_id;primaryKey;autoIncrement" json:"userId"`
	DeptID        *int64     `gorm:"column:dept_id" json:"deptId,omitempty"`
	UserName      string     `gorm:"column:user_name" json:"userName"`
	NickName      string     `gorm:"column:nick_name" json:"nickName"`
	UserType      string     `gorm:"column:user_type" json:"userType,omitempty"`
	Email         string     `gorm:"column:email" json:"email"`
	Phonenumber   string     `gorm:"column:phonenumber" json:"phonenumber"`
	Sex           string     `gorm:"column:sex" json:"sex"`
	Avatar        string     `gorm:"column:avatar" json:"avatar"`
	Status        string     `gorm:"column:status" json:"status"`    // char(1)：'0'正常 '1'停用
	DelFlag       string     `gorm:"column:del_flag" json:"delFlag"` // char(1)：'0'存在 '2'删除
	LoginIP       string     `gorm:"column:login_ip" json:"loginIp"`
	LoginDate     *time.Time `gorm:"column:login_date" json:"loginDate,omitempty"`
	PwdUpdateDate *time.Time `gorm:"column:pwd_update_date" json:"pwdUpdateDate,omitempty"`

	// DIFF(review): Java 的 SysUser 会被 Jackson 原样序列化，getInfo 响应里
	// 包含 password（BCrypt 散列）。Go 侧不返回该字段；前端
	// src/store/modules/user.js 与 src/views/** 均未读取 user.password，
	// 因此不影响契约，同时避免散列外泄。
	Password string `gorm:"column:password" json:"-"`

	// joinForeignKey / joinReferences 必须显式声明，否则 GORM 会按
	// "表名_字段名" 生成 sys_user_user_id 这类列名。
	Dept  *SysDept  `gorm:"foreignKey:DeptID;references:DeptID" json:"dept,omitempty"`
	Roles []SysRole `gorm:"many2many:sys_user_role;foreignKey:UserID;joinForeignKey:UserID;references:RoleID;joinReferences:RoleID" json:"roles,omitempty"`
}

// TableName 表名。
func (SysUser) TableName() string { return "sys_user" }

// IsAdmin 超级管理员判定，等价于 SecurityUtils.isAdmin(userId)：userId == 1。
func (u *SysUser) IsAdmin() bool { return u != nil && u.UserID == 1 }

// RoleKeys 返回角色标识集合（用于 hasRole / hasAnyRoles）。
func (u *SysUser) RoleKeys() []string {
	if u == nil || len(u.Roles) == 0 {
		return nil
	}
	keys := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		keys = append(keys, r.RoleKey)
	}
	return keys
}

// AfterFind 保证 roles 不为 nil（GORM 未 Preload 时为 nil，前端期望数组）。
//
// DIFF(review): Java 侧未加载时 roles 为 null，前端同样按 undefined 处理；
// Go 侧统一成空数组，避免 encoding/json 输出 null 造成前端遍历报错。
func (u *SysUser) AfterFind(_ *gorm.DB) error {
	if u.Roles == nil {
		u.Roles = []SysRole{}
	}
	return nil
}

// LoginUser 登录态，等价于 com.ruoyi.common.core.domain.model.LoginUser。
//
// 整体以 JSON 存入 Redis，key = login_tokens:<uuid>（CacheConstants.java:13）。
type LoginUser struct {
	UserID        int64    `json:"userId"`
	DeptID        *int64   `json:"deptId,omitempty"`
	Token         string   `json:"token"`     // uuid，Redis key 后缀
	LoginTime     int64    `json:"loginTime"` // 毫秒时间戳
	ExpireTime    int64    `json:"expireTime"`
	Ipaddr        string   `json:"ipaddr"`
	LoginLocation string   `json:"loginLocation"`
	Browser       string   `json:"browser"`
	Os            string   `json:"os"`
	Permissions   []string `json:"permissions"`
	User          *SysUser `json:"user"`
}

// IsAdmin 等价于 SecurityUtils.isAdmin(userId)。
func (l *LoginUser) IsAdmin() bool { return l != nil && l.UserID == 1 }

// HasPermission 等价于 PermissionService.hasPermissions(L155-158)：
// 持有 "*:*:*" 或目标权限即放行。
func (l *LoginUser) HasPermission(permission string) bool {
	if l == nil {
		return false
	}
	for _, p := range l.Permissions {
		if p == "*:*:*" || p == permission {
			return true
		}
	}
	return false
}
