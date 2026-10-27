package entity

import (
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model"
)

// SysRole 角色表 sys_role。
//
// data_scope / status / del_flag 均为 char(1)，一律用 string；
// menu_check_strictly / dept_check_strictly 为 tinyint(1)，用 bool。
type SysRole struct {
	model.BaseModel
	RoleID            int64  `gorm:"column:role_id;primaryKey;autoIncrement" json:"roleId"`
	RoleName          string `gorm:"column:role_name" json:"roleName"`
	RoleKey           string `gorm:"column:role_key" json:"roleKey"`
	RoleSort          int    `gorm:"column:role_sort" json:"roleSort"`
	DataScope         string `gorm:"column:data_scope" json:"dataScope"`
	MenuCheckStrictly bool   `gorm:"column:menu_check_strictly" json:"menuCheckStrictly"`
	DeptCheckStrictly bool   `gorm:"column:dept_check_strictly" json:"deptCheckStrictly"`
	Status            string `gorm:"column:status" json:"status"`
	DelFlag           string `gorm:"column:del_flag" json:"delFlag"`
}

// TableName 表名。
func (SysRole) TableName() string { return "sys_role" }

// IsAdmin 角色标识为 admin 即超管（对应 SysRole.isAdmin()）。
func (r *SysRole) IsAdmin() bool { return r != nil && r.RoleKey == "admin" }

// SysDept 部门表 sys_dept。
type SysDept struct {
	model.BaseModel
	DeptID    int64  `gorm:"column:dept_id;primaryKey;autoIncrement" json:"deptId"`
	ParentID  int64  `gorm:"column:parent_id" json:"parentId"`
	Ancestors string `gorm:"column:ancestors" json:"ancestors"`
	DeptName  string `gorm:"column:dept_name" json:"deptName"`
	OrderNum  int    `gorm:"column:order_num" json:"orderNum"`
	Leader    string `gorm:"column:leader" json:"leader"`
	Phone     string `gorm:"column:phone" json:"phone"`
	Email     string `gorm:"column:email" json:"email"`
	Status    string `gorm:"column:status" json:"status"`
	DelFlag   string `gorm:"column:del_flag" json:"delFlag"`
}

// TableName 表名。
func (SysDept) TableName() string { return "sys_dept" }

// SysConfig 参数配置表 sys_config。
type SysConfig struct {
	model.BaseModel
	ConfigID    int64  `gorm:"column:config_id;primaryKey;autoIncrement" json:"configId"`
	ConfigName  string `gorm:"column:config_name" json:"configName"`
	ConfigKey   string `gorm:"column:config_key" json:"configKey"`
	ConfigValue string `gorm:"column:config_value" json:"configValue"`
	ConfigType  string `gorm:"column:config_type" json:"configType"` // char(1)：Y 系统内置
}

// TableName 表名。
func (SysConfig) TableName() string { return "sys_config" }

// SysLogininfor 系统访问记录 sys_logininfor。
//
// 注意：该表没有 create_time / update_time，只有 login_time（sql:563-576），
// 因此不嵌入 BaseModel。status 为 char(1)：'0'成功 / '1'失败。
type SysLogininfor struct {
	InfoID        int64     `gorm:"column:info_id;primaryKey;autoIncrement" json:"infoId"`
	UserName      string    `gorm:"column:user_name" json:"userName"`
	Ipaddr        string    `gorm:"column:ipaddr" json:"ipaddr"`
	LoginLocation string    `gorm:"column:login_location" json:"loginLocation"`
	Browser       string    `gorm:"column:browser" json:"browser"`
	Os            string    `gorm:"column:os" json:"os"`
	Status        string    `gorm:"column:status" json:"status"`
	Msg           string    `gorm:"column:msg" json:"msg"`
	LoginTime     time.Time `gorm:"column:login_time" json:"loginTime"`
}

// TableName 表名。
func (SysLogininfor) TableName() string { return "sys_logininfor" }
