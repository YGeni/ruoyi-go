package service

import (
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"
)

// ---------- 登录日志：对应 SysLogininforServiceImpl.java ----------

// SelectLogininforList 分页查询，等价 SysLogininforServiceImpl.java:30-34。
func SelectLogininforList(q *query.LogininforList, p *page.PageDomain) ([]entity.SysLogininfor, int64, error) {
	return repository.SelectLogininforList(q, p)
}

// DeleteLogininforByIDs 批量删除，等价 SysLogininforServiceImpl.java:59-63。
func DeleteLogininforByIDs(infoIDs []int64) (int64, error) {
	return repository.DeleteLogininforByIDs(infoIDs)
}

// CleanLogininfor 清空，等价 SysLogininforServiceImpl.java:71-74。
func CleanLogininfor() error {
	return repository.CleanLogininfor()
}

// UnlockLogininfor 解锁账号，等价 SysUserOnline 侧的
// SysPasswordService.clearLoginRecordCache（SysPasswordService.java:57-63）：
// 删掉 Redis 里的 pwd_err_cnt:<username> 失败计数。
//
// 注意 SysLogininforController.java:74-81 只调了这一个方法，
// **不动 sys_user.status**（账号停用是另一回事）。
func UnlockLogininfor(userName string) error {
	if userName == "" {
		return nil
	}
	return securityx.ClearLoginRecordCache(userName)
}

// ---------- 操作日志：对应 SysOperLogServiceImpl.java ----------

// SelectOperLogList 分页查询，等价 SysOperLogServiceImpl.java:40-44。
func SelectOperLogList(q *query.OperLogList, p *page.PageDomain) ([]entity.SysOperLog, int64, error) {
	return repository.SelectOperLogList(q, p)
}

// DeleteOperLogByIDs 批量删除，等价 SysOperLogServiceImpl.java:89-93。
func DeleteOperLogByIDs(operIDs []int64) (int64, error) {
	return repository.DeleteOperLogByIDs(operIDs)
}

// CleanOperLog 清空，等价 SysOperLogServiceImpl.java:101-104。
func CleanOperLog() error {
	return repository.CleanOperLog()
}

// ---------- 导出 ----------

// LogininforExportRows 登录日志导出的表头与数据行。
//
// 列名与顺序取自 SysLogininfor.java 的 @Excel(name = "...") 注解。
func LogininforExportRows(rows []entity.SysLogininfor) ([]string, [][]any) {
	headers := []string{"序号", "用户账号", "登录状态", "登录地址", "登录地点", "浏览器", "操作系统", "提示消息", "访问时间"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		status := "成功"
		if r.Status == LoginStatusFail {
			status = "失败"
		}
		out = append(out, []any{r.InfoID, r.UserName, status, r.Ipaddr,
			r.LoginLocation, r.Browser, r.Os, r.Msg,
			r.LoginTime.Format("2006-01-02 15:04:05")})
	}
	return headers, out
}

// OperLogExportRows 操作日志导出的表头与数据行。
//
// 列名与顺序取自 SysOperLog.java 的 @Excel(name = "...") 注解；
// businessType / operatorType / status 的字典翻译同样取自 @Excel 的
// readConverterExp（"0=其它,1=新增,..."）。
func OperLogExportRows(rows []entity.SysOperLog) ([]string, [][]any) {
	headers := []string{"操作序号", "操作模块", "业务类型", "请求方法", "请求方式", "操作类别",
		"操作人员", "部门名称", "请求地址", "操作地址", "操作地点", "请求参数", "返回参数",
		"状态", "错误消息", "操作时间", "消耗时间"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{
			r.OperID,
			r.Title,
			businessTypeName(r.BusinessType),
			r.Method,
			r.RequestMethod,
			operatorTypeName(r.OperatorType),
			r.OperName,
			r.DeptName,
			r.OperUrl,
			r.OperIp,
			r.OperLocation,
			r.OperParam,
			r.JsonResult,
			operStatusName(r.Status),
			r.ErrorMsg,
			r.OperTime.Format("2006-01-02 15:04:05"),
			r.CostTime,
		})
	}
	return headers, out
}

// 以下三个字典翻译对应 @Excel 的 readConverterExp。
func businessTypeName(v *int) string {
	if v == nil {
		return ""
	}
	switch *v {
	case 0:
		return "其它"
	case 1:
		return "新增"
	case 2:
		return "修改"
	case 3:
		return "删除"
	case 4:
		return "授权"
	case 5:
		return "导出"
	case 6:
		return "导入"
	case 7:
		return "强退"
	case 8:
		return "生成代码"
	case 9:
		return "清空数据"
	}
	return ""
}

func operatorTypeName(v *int) string {
	if v == nil {
		return ""
	}
	switch *v {
	case 0:
		return "其它"
	case 1:
		return "后台用户"
	case 2:
		return "手机端用户"
	}
	return ""
}

func operStatusName(v *int) string {
	if v == nil {
		return ""
	}
	if *v == 0 {
		return "正常"
	}
	return "异常"
}
