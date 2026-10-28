// Package query 定义各模块列表接口的查询条件，等价于 Java Controller 里
// 直接以实体（如 SysUser user）接收的 query/form 绑定参数。
//
// 契约来源：
//   - 分页参数（pageNum/pageSize/orderByColumn/isAsc/reasonable）**不在这里**，
//     由 pkg/page.FromRequest 单独解析（TableSupport.java:16-49）。
//   - 日期范围来自前端 addDateRange（F/src/utils/ruoyi.js:54-66）设置到
//     search.params.beginTime / endTime，再经 tansParams 展平成
//     query key 字面量 "params[beginTime]" / "params[endTime]"。
//   - searchValue 来自 BaseEntity.searchValue（BaseEntity.java:22）。
package query

import (
	"github.com/gin-gonic/gin"
)

// 前端 tansParams 展平后的字面 query key（含方括号，必须原样读取）。
const (
	KeyBeginTime = "params[beginTime]"
	KeyEndTime   = "params[endTime]"
)

// BaseList 列表公共条件。
type BaseList struct {
	SearchValue string `form:"searchValue" json:"searchValue"`
	BeginTime   string `form:"-" json:"-"`
	EndTime     string `form:"-" json:"-"`
}

// FillDateRange 取日期范围。
//
// DIFF(review): gin 的 form 绑定不支持 "params[beginTime]" 这种方括号嵌套 key，
// 因此这里直接按字面 key 从 query 读取；语义与 Java 的 params.beginTime 一致。
//
// 导出接口是 **POST + application/x-www-form-urlencoded**（request.js:127-133 的
// download 用 tansParams 把参数放进了请求体），拿不到 query，因此再兜一层 PostForm。
func (b *BaseList) FillDateRange(c *gin.Context) {
	if b == nil {
		return
	}
	b.BeginTime = c.Query(KeyBeginTime)
	b.EndTime = c.Query(KeyEndTime)
	if b.BeginTime == "" {
		b.BeginTime = c.PostForm(KeyBeginTime)
	}
	if b.EndTime == "" {
		b.EndTime = c.PostForm(KeyEndTime)
	}
}

// HasDateRange 是否传了日期范围。
func (b *BaseList) HasDateRange() bool {
	return b != nil && (b.BeginTime != "" || b.EndTime != "")
}

// UserList GET /system/user/list 的查询条件。
//
// 对应 SysUserMapper.xml:59-83 的 selectUserList 各 if 分支：
// userId / userName / status / phonenumber / params.beginTime / params.endTime / deptId。
//
// DIFF(review): searchValue 在 Java 的 selectUserList 里**没有**对应的 if 分支
// （SysUserMapper.xml 实测无 searchValue），按 1b 验收要求在此补上，
// 过滤口径与 3.9 的 SysNoticeReadMapper.xml:79-83 保持一致：
// user_name like %v% or nick_name like %v%。不传该参数时行为与 Java 完全一致。
type UserList struct {
	BaseList
	UserID      int64  `form:"userId" json:"userId"`
	UserName    string `form:"userName" json:"userName"`
	Status      string `form:"status" json:"status"`
	Phonenumber string `form:"phonenumber" json:"phonenumber"`
	DeptID      int64  `form:"deptId" json:"deptId"`
}

// AuthUserList GET /system/role/authUser/allocatedList | unallocatedList 的查询条件。
//
// 对应 SysUserMapper.xml:85-121：userName / phonenumber（无 status、无 userId）。
type AuthUserList struct {
	BaseList
	UserName    string `form:"userName" json:"userName"`
	Phonenumber string `form:"phonenumber" json:"phonenumber"`
}

// RoleList GET /system/role/list 的查询条件。
//
// 对应 SysRoleMapper.xml:36-61：roleId / roleName / status / roleKey / 日期范围。
// 排序固定 order by r.role_sort（Java 写在 SQL 末尾，不来自前端）。
type RoleList struct {
	BaseList
	RoleID   int64  `form:"roleId" json:"roleId"`
	RoleName string `form:"roleName" json:"roleName"`
	RoleKey  string `form:"roleKey" json:"roleKey"`
	Status   string `form:"status" json:"status"`
}

// DeptList GET /system/dept/list 的查询条件。
//
// 对应 SysDeptMapper.xml:35-49：deptId / parentId / deptName / status。
// 排序固定 order by d.parent_id, d.order_num。
type DeptList struct {
	DeptID   int64  `form:"deptId" json:"deptId"`
	ParentID int64  `form:"parentId" json:"parentId"`
	DeptName string `form:"deptName" json:"deptName"`
	Status   string `form:"status" json:"status"`
}

// PostList GET /system/post/list 的查询条件。
//
// 对应 SysPostMapper.xml：postCode / postName / status。
type PostList struct {
	PostCode string `form:"postCode" json:"postCode"`
	PostName string `form:"postName" json:"postName"`
	Status   string `form:"status" json:"status"`
}

// DictTypeList GET /system/dict/type/list 的查询条件。
//
// 对应 SysDictTypeMapper.xml：dictName / dictType / status。
type DictTypeList struct {
	DictName string `form:"dictName" json:"dictName"`
	DictType string `form:"dictType" json:"dictType"`
	Status   string `form:"status" json:"status"`
}

// DictDataList GET /system/dict/data/list 的查询条件。
//
// 对应 SysDictDataMapper.xml：dictType / dictLabel / status。
type DictDataList struct {
	DictType  string `form:"dictType" json:"dictType"`
	DictLabel string `form:"dictLabel" json:"dictLabel"`
	Status    string `form:"status" json:"status"`
}

// ConfigList GET /system/config/list 的查询条件。
//
// 对应 SysConfigMapper.xml：configName / configKey / configType。
type ConfigList struct {
	ConfigName string `form:"configName" json:"configName"`
	ConfigKey  string `form:"configKey" json:"configKey"`
	ConfigType string `form:"configType" json:"configType"`
}

// NoticeList GET /system/notice/list 的查询条件。
//
// 对应 SysNoticeMapper.xml:30-41：noticeTitle / noticeType / createBy。
type NoticeList struct {
	NoticeTitle string `form:"noticeTitle" json:"noticeTitle"`
	NoticeType  string `form:"noticeType" json:"noticeType"`
	CreateBy    string `form:"createBy" json:"createBy"`
}

// ReadUserList GET /system/notice/readUsers/list 的查询条件。
//
// 对应 SysNoticeReadMapper.xml:73-95：noticeId（必填）+ searchValue。
// MenuList 菜单查询条件，等价 SysMenuMapper.xml:38-47 的三个动态条件。
//
// 注意 /system/menu/list **不分页**（Java 的 list() 没有 startPage()），
// 前端 src/views/system/menu/index.vue:346 直接 handleTree(response.data)。
type MenuList struct {
	MenuName string `form:"menuName" json:"menuName"`
	Visible  string `form:"visible" json:"visible"`
	Status   string `form:"status" json:"status"`
}

type ReadUserList struct {
	NoticeID int64 `form:"noticeId" json:"noticeId"`
	BaseList       // searchValue
}

// LogininforList GET /monitor/logininfor/list 的查询条件。
//
// 对应 SysLogininforMapper.xml 的 selectLogininforList 各 if 分支：
// ipaddr like / status = / user_name like / params.beginTime / params.endTime，
// 默认 order by info_id desc。
type LogininforList struct {
	BaseList
	Ipaddr   string `form:"ipaddr" json:"ipaddr"`
	UserName string `form:"userName" json:"userName"`
	Status   string `form:"status" json:"status"`
}

// OperLogList GET /monitor/operlog/list 的查询条件。
//
// 对应 SysOperLogMapper.xml 的 selectOperLogList 各 if 分支：
// oper_ip like / title like / business_type = / status = / oper_name like
// / params.beginTime / params.endTime，默认 order by oper_id desc。
//
// DIFF(review): XML 里还有一个 businessTypes（数组，IN 查询）分支，
// 前端 operlog 页面只提交单个 businessType（src/views/monitor/operlog/index.vue:33），
// 该分支不实现。
type OperLogList struct {
	BaseList
	OperIp       string `form:"operIp" json:"operIp"`
	Title        string `form:"title" json:"title"`
	BusinessType *int   `form:"businessType" json:"businessType"`
	Status       *int   `form:"status" json:"status"`
	OperName     string `form:"operName" json:"operName"`
}
