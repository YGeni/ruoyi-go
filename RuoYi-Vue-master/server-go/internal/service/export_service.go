package service

// 本文件是各模块「导出」的列构建，等价 Java domain 上的 @Excel(name = "...")
// 注解 —— 注解声明"导出哪几列、表头叫什么、字典怎么翻译"，Java 里由
// ExcelUtil 反射读取，Go 侧没有注解，就在这里显式写出来。
//
// 表名与列顺序跟 Java 导出保持一致，这样导出的 xlsx 能直接被 Java 的
// 导入功能读回去（列是按表头名匹配的）。
//
// 这批导出原本在二期清单里（permRoutesPhase2），现已补齐。

import (
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
)

// ConfigExportRows POST /system/config/export 的表头与数据行。
func ConfigExportRows(rows []entity.SysConfig) ([]string, [][]any) {
	headers := []string{"参数序号", "参数名称", "参数键名", "参数键值", "系统内置", "备注"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		builtin := "否"
		if r.ConfigType == "Y" {
			builtin = "是"
		}
		out = append(out, []any{r.ConfigID, r.ConfigName, r.ConfigKey, r.ConfigValue, builtin, r.Remark})
	}
	return headers, out
}

// DictTypeExportRows POST /system/dict/type/export 的表头与数据行。
func DictTypeExportRows(rows []entity.SysDictType) ([]string, [][]any) {
	headers := []string{"字典序号", "字典名称", "字典类型", "状态", "备注"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.DictID, r.DictName, r.DictType, statusText(r.Status), r.Remark})
	}
	return headers, out
}

// DictDataExportRows POST /system/dict/data/export 的表头与数据行。
func DictDataExportRows(rows []entity.SysDictData) ([]string, [][]any) {
	headers := []string{"字典编码", "字典排序", "字典标签", "字典键值", "字典类型", "样式属性", "表格回显样式", "是否默认", "状态", "备注"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		isDefault := "否"
		if r.IsDefault == "Y" {
			isDefault = "是"
		}
		out = append(out, []any{r.DictCode, r.DictSort, r.DictLabel, r.DictValue, r.DictType,
			r.CssClass, r.ListClass, isDefault, statusText(r.Status), r.Remark})
	}
	return headers, out
}

// PostExportRows POST /system/post/export 的表头与数据行。
func PostExportRows(rows []entity.SysPost) ([]string, [][]any) {
	headers := []string{"岗位序号", "岗位编码", "岗位名称", "岗位排序", "状态", "备注"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.PostID, r.PostCode, r.PostName, r.PostSort, statusText(r.Status), r.Remark})
	}
	return headers, out
}

// RoleExportRows POST /system/role/export 的表头与数据行。
func RoleExportRows(rows []entity.SysRole) ([]string, [][]any) {
	headers := []string{"角色序号", "角色名称", "角色权限", "角色排序", "状态", "备注"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.RoleID, r.RoleName, r.RoleKey, r.RoleSort, statusText(r.Status), r.Remark})
	}
	return headers, out
}

// UserExportHeaders 用户导出的表头。
//
// 与导入模板 ImportUserHeaders **不是同一套**：导出带"用户序号""最后登录IP"
// 这类只读列，导入按 ImportUserHeaders 的顺序取值。
var UserExportHeaders = []string{"用户序号", "登录名称", "用户名称", "用户邮箱", "手机号码", "用户性别", "帐号状态", "最后登录IP", "最后登录时间"}

// UserExportRows POST /system/user/export 的表头与数据行。
//
// 用户性别/帐号状态在 Java 侧是 @Excel 的 readConverterExp
// （"0=男,1=女,2=未知" / "0=正常,1=停用"），这里按同样的映射翻译。
func UserExportRows(rows []entity.SysUser) ([]string, [][]any) {
	headers := UserExportHeaders
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.UserID, r.UserName, r.NickName, r.Email, r.Phonenumber,
			sexText(r.Sex), statusText(r.Status), r.LoginIP, fmtTime(r.LoginDate)})
	}
	return headers, out
}

// statusText '0'正常 '1'停用。
func statusText(s string) string {
	if s == config.Disable {
		return "停用"
	}
	return "正常"
}

// sexText '0'男 '1'女 '2'未知。
func sexText(s string) string {
	switch s {
	case "0":
		return "男"
	case "1":
		return "女"
	default:
		return "未知"
	}
}

// fmtTime 导出时的时间格式；空值返回空串（不是 "0001-01-01"）。
func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
