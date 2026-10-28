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

	// Permissions 角色菜单权限集，等价于 SysRole.permissions（SysRole.java:65）。
	//
	// 该字段**必须存在**：DataScopeAspect.java:69 的
	// StringUtils.containsAny(role.getPermissions(), Convert.toStrArray(permission))
	// 依赖它做角色级权限筛选。若缺失，任何带 permission 的 @DataScope 调用
	// 都会把所有角色跳过 -> conditions 为空 -> 落到 "OR d.dept_id = 0" 兜底 -> 返回 0 行。
	// 由 SysPermissionService.java:77（role.setPermissions）在登录时填充，
	// Go 侧对应 service.GetMenuPermission。
	//
	// DIFF(review): Java 侧该字段无 @JsonIgnore，未填充时会输出 "permissions":null；
	// Go 侧用 omitempty，未填充时不下发该字段（前端不读 roles[].permissions）。
	Permissions []string `gorm:"-" json:"permissions,omitempty"`

	// MenuIds 非持久化字段，等价 SysRole.menuIds（SysRole.java:60 之后）：
	// 前端新增/修改角色时用 menuIds 数组提交菜单授权，后端据此重写 sys_role_menu。
	MenuIds []int64 `gorm:"-" json:"menuIds,omitempty"`
}

// TableName 表名。
func (SysRole) TableName() string { return "sys_role" }

// IsAdmin 角色标识为 admin 即超管（对应 SysRole.isAdmin()）。
func (r *SysRole) IsAdmin() bool { return r != nil && r.RoleKey == "admin" }

// HasPermission 等价于 StringUtils.containsAny(role.getPermissions(), perms...)
// （StringUtils.java:456-473）：permission 为空时按 Java 语义由调用方短路，
// 这里只负责"角色是否持有给定权限中的任意一个"。
func (r *SysRole) HasPermission(perms ...string) bool {
	if r == nil || len(r.Permissions) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(r.Permissions))
	for _, p := range r.Permissions {
		set[p] = struct{}{}
	}
	for _, p := range perms {
		if _, ok := set[p]; ok {
			return true
		}
	}
	return false
}

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

// SysJob 定时任务调度表 sys_job。
//
// 该表没有 del_flag（删除是真删），create_time / update_time 是普通列，
// 与 BaseModel 的语义一致，因此照常嵌入。
type SysJob struct {
	model.BaseModel
	JobID          int64  `gorm:"column:job_id;primaryKey;autoIncrement" json:"jobId"`
	JobName        string `gorm:"column:job_name" json:"jobName"`
	JobGroup       string `gorm:"column:job_group" json:"jobGroup"`
	InvokeTarget   string `gorm:"column:invoke_target" json:"invokeTarget"`
	CronExpression string `gorm:"column:cron_expression" json:"cronExpression"`
	MisfirePolicy  string `gorm:"column:misfire_policy" json:"misfirePolicy"`
	Concurrent     string `gorm:"column:concurrent" json:"concurrent"`
	Status         string `gorm:"column:status" json:"status"`
}

// TableName 表名。
func (SysJob) TableName() string { return "sys_job" }

// SysJobLog 定时任务调度日志表 sys_job_log。
//
// 与 SysJob 不同：这张表**只有 create_time、没有 update_time**，
// 所以不嵌 BaseModel（否则 GORM 会去写不存在的 update_time 列）。
type SysJobLog struct {
	JobLogID      int64      `gorm:"column:job_log_id;primaryKey;autoIncrement" json:"jobLogId"`
	JobName       string     `gorm:"column:job_name" json:"jobName"`
	JobGroup      string     `gorm:"column:job_group" json:"jobGroup"`
	InvokeTarget  string     `gorm:"column:invoke_target" json:"invokeTarget"`
	JobMessage    string     `gorm:"column:job_message" json:"jobMessage"`
	Status        string     `gorm:"column:status" json:"status"`
	ExceptionInfo string     `gorm:"column:exception_info" json:"exceptionInfo"`
	StartTime     *time.Time `gorm:"column:start_time" json:"startTime"`
	EndTime       *time.Time `gorm:"column:end_time" json:"endTime"`
	CreateTime    *time.Time `gorm:"column:create_time" json:"createTime"`
}

// TableName 表名。
func (SysJobLog) TableName() string { return "sys_job_log" }

// SysOperLog 操作日志记录 sys_oper_log。
//
// 与 SysLogininfor 同理：该表没有 create_time / update_time，只有 oper_time，
// 因此不嵌入 BaseModel。
type SysOperLog struct {
	OperID        int64     `gorm:"column:oper_id;primaryKey;autoIncrement" json:"operId"`
	Title         string    `gorm:"column:title" json:"title"`
	BusinessType  *int      `gorm:"column:business_type" json:"businessType"`
	Method        string    `gorm:"column:method" json:"method"`
	RequestMethod string    `gorm:"column:request_method" json:"requestMethod"`
	OperatorType  *int      `gorm:"column:operator_type" json:"operatorType"`
	OperName      string    `gorm:"column:oper_name" json:"operName"`
	DeptName      string    `gorm:"column:dept_name" json:"deptName"`
	OperUrl       string    `gorm:"column:oper_url" json:"operUrl"`
	OperIp        string    `gorm:"column:oper_ip" json:"operIp"`
	OperLocation  string    `gorm:"column:oper_location" json:"operLocation"`
	OperParam     string    `gorm:"column:oper_param" json:"operParam"`
	JsonResult    string    `gorm:"column:json_result" json:"jsonResult"`
	Status        *int      `gorm:"column:status" json:"status"`
	ErrorMsg      string    `gorm:"column:error_msg" json:"errorMsg"`
	OperTime      time.Time `gorm:"column:oper_time" json:"operTime"`
	CostTime      int64     `gorm:"column:cost_time" json:"costTime"`
}

// TableName 表名。
func (SysOperLog) TableName() string { return "sys_oper_log" }

// SysPost 岗位信息表 sys_post。
type SysPost struct {
	model.BaseModel
	PostID   int64  `gorm:"column:post_id;primaryKey;autoIncrement" json:"postId"`
	PostCode string `gorm:"column:post_code" json:"postCode"`
	PostName string `gorm:"column:post_name" json:"postName"`
	PostSort int    `gorm:"column:post_sort" json:"postSort"`
	Status   string `gorm:"column:status" json:"status"` // char(1)：'0'正常 '1'停用
}

// TableName 表名。
func (SysPost) TableName() string { return "sys_post" }

// SysDictType 字典类型表 sys_dict_type（dict_type 有唯一约束）。
type SysDictType struct {
	model.BaseModel
	DictID   int64  `gorm:"column:dict_id;primaryKey;autoIncrement" json:"dictId"`
	DictName string `gorm:"column:dict_name" json:"dictName"`
	DictType string `gorm:"column:dict_type" json:"dictType"`
	Status   string `gorm:"column:status" json:"status"` // char(1)：'0'正常 '1'停用
}

// TableName 表名。
func (SysDictType) TableName() string { return "sys_dict_type" }

// SysDictData 字典数据表 sys_dict_data。
//
// is_default 为 char(1)：'Y' 是默认 / 'N' 否，一律 string（全局规则：char(1) 不用 bool）。
type SysDictData struct {
	model.BaseModel
	DictCode  int64  `gorm:"column:dict_code;primaryKey;autoIncrement" json:"dictCode"`
	DictSort  int    `gorm:"column:dict_sort" json:"dictSort"`
	DictLabel string `gorm:"column:dict_label" json:"dictLabel"`
	DictValue string `gorm:"column:dict_value" json:"dictValue"`
	DictType  string `gorm:"column:dict_type" json:"dictType"`
	CssClass  string `gorm:"column:css_class" json:"cssClass"`
	ListClass string `gorm:"column:list_class" json:"listClass"`
	IsDefault string `gorm:"column:is_default" json:"isDefault"`
	Status    string `gorm:"column:status" json:"status"`
}

// TableName 表名。
func (SysDictData) TableName() string { return "sys_dict_data" }

// SysNotice 通知公告表 sys_notice。
//
// notice_content 在 DDL 里是 longblob，但 Java 侧 SysNotice.noticeContent 是 String
// （MyBatis 负责 longblob <-> String 转换），前端亦按字符串消费，故 Go 侧同样用 string。
type SysNotice struct {
	model.BaseModel
	NoticeID      int64  `gorm:"column:notice_id;primaryKey;autoIncrement" json:"noticeId"`
	NoticeTitle   string `gorm:"column:notice_title" json:"noticeTitle"`
	NoticeType    string `gorm:"column:notice_type" json:"noticeType"` // char(1)：'1'通知 '2'公告
	NoticeContent string `gorm:"column:notice_content" json:"noticeContent"`
	Status        string `gorm:"column:status" json:"status"` // char(1)：'0'正常 '1'关闭
}

// TableName 表名。
func (SysNotice) TableName() string { return "sys_notice" }

// SysNoticeRead 公告已读记录表 sys_notice_read。
//
// 该表**无任何时间字段之外的审计列**，且唯一约束 uk_user_notice(user_id, notice_id)，
// 同一用户同一公告只记录一次（对应 markRead 的幂等写入）。
// 不嵌入 BaseModel（表内无 create_by/create_time/update_by/update_time/remark）。
type SysNoticeRead struct {
	ReadID   int64      `gorm:"column:read_id;primaryKey;autoIncrement" json:"readId"`
	NoticeID int64      `gorm:"column:notice_id" json:"noticeId"`
	UserID   int64      `gorm:"column:user_id" json:"userId"`
	ReadTime *time.Time `gorm:"column:read_time" json:"readTime,omitempty"`
}

// TableName 表名。
func (SysNoticeRead) TableName() string { return "sys_notice_read" }

// SysNoticeWithRead 带当前用户已读标记的公告（3.9 新增）。
//
// 等价 SysNoticeReadMapper.xml:44-52 的 selectNoticeListWithReadStatus
// （resultType=SysNotice，isRead 由 left join sys_notice_read 计算出）。
//
// DIFF(review): Java 复用 SysNotice 实体并新增非持久化 isRead 字段；
// Go 侧单独定义 DTO，避免给持久化实体加数据库不存在的列。
type SysNoticeWithRead struct {
	NoticeID    int64      `json:"noticeId"`
	NoticeTitle string     `json:"noticeTitle"`
	NoticeType  string     `json:"noticeType"`
	Status      string     `json:"status"`
	CreateBy    string     `json:"createBy"`
	CreateTime  *time.Time `json:"createTime,omitempty"`
	IsRead      bool       `json:"isRead"`
}

// NoticeReadUser 已读某公告的用户（3.9 新增，/system/notice/readUsers/list 的行结构）。
//
// 等价 SysNoticeReadMapper.xml:73-95 的 resultType=java.util.Map 的六个字段。
type NoticeReadUser struct {
	UserID      int64      `json:"userId"`
	UserName    string     `json:"userName"`
	NickName    string     `json:"nickName"`
	DeptName    string     `json:"deptName"`
	Phonenumber string     `json:"phonenumber"`
	ReadTime    *time.Time `json:"readTime,omitempty"`
}

// 以下三张关联表**无任何时间字段**（sql/ry_20260417.sql），因此都不嵌入 BaseModel。
// 用途：数据权限自定义分支要查 sys_role_dept；授权角色要写 sys_user_role。

// SysUserRole 用户与角色关联表 sys_user_role。
type SysUserRole struct {
	UserID int64 `gorm:"column:user_id;primaryKey" json:"userId"`
	RoleID int64 `gorm:"column:role_id;primaryKey" json:"roleId"`
}

// TableName 表名。
func (SysUserRole) TableName() string { return "sys_user_role" }

// SysRoleDept 角色与部门关联表 sys_role_dept（数据权限"自定义"分支的数据源）。
type SysRoleDept struct {
	RoleID int64 `gorm:"column:role_id;primaryKey" json:"roleId"`
	DeptID int64 `gorm:"column:dept_id;primaryKey" json:"deptId"`
}

// TableName 表名。
func (SysRoleDept) TableName() string { return "sys_role_dept" }

// SysRoleMenu 角色与菜单关联表 sys_role_menu。
type SysRoleMenu struct {
	RoleID int64 `gorm:"column:role_id;primaryKey" json:"roleId"`
	MenuID int64 `gorm:"column:menu_id;primaryKey" json:"menuId"`
}

// TableName 表名。
func (SysRoleMenu) TableName() string { return "sys_role_menu" }
