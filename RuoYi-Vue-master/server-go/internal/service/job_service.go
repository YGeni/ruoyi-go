package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/internal/scheduler"
	"github.com/YGeni/ruoyi-go/server-go/pkg/cronx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// 校验失败文案，逐字照抄/ 118-147。
const (
	MsgCronInvalid     = "Cron表达式不正确"
	MsgNoRMI           = "目标字符串不允许'rmi'调用"
	MsgNoLDAP          = "目标字符串不允许'ldap(s)'调用"
	MsgNoHTTP          = "目标字符串不允许'http(s)'调用"
	MsgIllegalTarget   = "目标字符串存在违规"
	MsgNotInWhiteList  = "目标字符串不在白名单内"
	MsgJobNotFound     = "任务不存在或已过期！"
	MsgJobLogCleanDone = "清除成功"
)

// ---------- 定时任务：对应----------

// SelectJobList 分页查询，等价。
func SelectJobList(q *query.JobList, p *page.PageDomain) ([]entity.SysJob, int64, error) {
	return repository.SelectJobList(q, p)
}

// SelectJobByID 详情，等价。
func SelectJobByID(jobID int64) (*entity.SysJob, error) {
	return repository.SelectJobByID(jobID)
}

// InsertJob 新增，等价。
//
// 校验链照抄 SysJobController.add(L82-112)，返回 error 由 handler 转成 error(msg)。
func InsertJob(j *entity.SysJob) error {
	if err := validateJob(j, "新增"); err != nil {
		return err
	}
	j.Status = stringOrDefault(j.Status, config.Normal)
	j.MisfirePolicy = stringOrDefault(j.MisfirePolicy, "3")
	j.Concurrent = stringOrDefault(j.Concurrent, "1")
	if err := repository.InsertJob(j); err != nil {
		return err
	}
	return scheduler.Reload()
}

// UpdateJob 修改，等价。
func UpdateJob(j *entity.SysJob) error {
	if err := validateJob(j, "修改"); err != nil {
		return err
	}
	if err := repository.UpdateJob(j); err != nil {
		return err
	}
	return scheduler.Reload()
}

// ChangeStatus 启停，等价。
//
// Java 侧是 `newJob.setStatus(job.getStatus())` 后整体 updateJob；
// Go 侧只改 status 一个字段（quartz 的 resume/pause 也只动状态）。
func ChangeStatus(j *entity.SysJob) error {
	if err := repository.UpdateJobStatus(j); err != nil {
		return err
	}
	return scheduler.Refresh(j.JobID)
}

// Run 立即执行一次，等价。
//
// Java 是 `scheduler.triggerJob(...)`，任务不存在时抛 TaskException，
// Controller 捕获后返回 error("任务不存在或已过期！")。
// Go 侧没有 quartz，直接按 registry 执行并写日志。
func Run(j *entity.SysJob) error {
	cur, err := repository.SelectJobByID(j.JobID)
	if err != nil {
		return err
	}
	if cur == nil {
		return errors.New(MsgJobNotFound)
	}
	start := time.Now()
	msg, runErr := scheduler.Invoke(cur.InvokeTarget)
	end := time.Now()
	l := &entity.SysJobLog{
		JobName:      cur.JobName,
		JobGroup:     cur.JobGroup,
		InvokeTarget: cur.InvokeTarget,
		StartTime:    &start,
		EndTime:      &end,
		CreateTime:   &start,
		Status:       "0",
	}
	if runErr != nil {
		l.Status = "1"
		l.ExceptionInfo = runErr.Error()
		l.JobMessage = "执行失败"
	} else {
		l.JobMessage = msg + " 总共耗时：" + end.Sub(start).String()
	}
	return repository.InsertJobLog(l)
}

// DeleteJobByIDs 批量删除，等价。
func DeleteJobByIDs(jobIDs []int64) error {
	if _, err := repository.DeleteJobByIDs(jobIDs); err != nil {
		return err
	}
	for _, id := range jobIDs {
		scheduler.Remove(id)
	}
	return nil
}

// validateJob 校验 cron 与调用目标，逐条对应 SysJobController.add 的六个分支。
//
// op 是"新增"或"修改"（文案前缀不同）。
func validateJob(j *entity.SysJob, op string) error {
	if !cronx.IsValid(j.CronExpression) {
		return errors.New(fmt.Sprintf("%s任务'%s'失败，%s", op, j.JobName, MsgCronInvalid))
	}
	t := j.InvokeTarget
	prefix := func(msg string) error {
		return errors.New(fmt.Sprintf("%s任务'%s'失败，%s", op, j.JobName, msg))
	}
	lower := strings.ToLower(t)
	if strings.Contains(lower, "rmi:") {
		return prefix(MsgNoRMI)
	}
	if strings.Contains(lower, "ldap:") || strings.Contains(lower, "ldaps:") {
		return prefix(MsgNoLDAP)
	}
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		return prefix(MsgNoHTTP)
	}
	for _, bad := range jobErrorStr {
		if strings.Contains(lower, strings.ToLower(bad)) {
			return prefix(MsgIllegalTarget)
		}
	}
	if !whiteList(t) {
		return prefix(MsgNotInWhiteList)
	}
	return nil
}

// jobErrorStr 对应 Constants.JOB_ERROR_STR。
var jobErrorStr = []string{
	"java.net.URL", "javax.naming.InitialContext", "org.yaml.snakeyaml",
	"org.springframework", "com.sun.org.apache.", "java.lang.Runtime",
	"java.lang.Class", "java.lang.ClassLoader", "java.lang.System",
	"java.lang.Thread", "java.lang.ProcessBuilder", "javax.script.",
	"groovy.lang.", "java.io.File", "java.nio.file.", "sun.misc.",
}

// whiteList 等价 ScheduleUtils.whiteList。
//
// Java 的逻辑是：invokeTarget 里 "(" 之前的包名含超过 1 个点时，
// 必须以前缀常量开头；否则取 bean 名查 Spring 容器再判包名。
// Go 侧没有 Spring，直接按"前缀必须是 com.ruoyi.quartz.task"判定，
// 落点与 Java 对 RuoYi 自带任务的结果一致。
func whiteList(invokeTarget string) bool {
	pkg := invokeTarget
	if i := strings.Index(invokeTarget, "("); i >= 0 {
		pkg = invokeTarget[:i]
	}
	if strings.Count(pkg, ".") > 1 {
		for _, w := range []string{config.JobWhitelistStr} {
			if strings.HasPrefix(invokeTarget, w) {
				return true
			}
		}
		return false
	}
	// 单个点的形式（ryTask.xxx）：等价于查到 bean 后判其包名
	bean := strings.Split(invokeTarget, ".")[0]
	return scheduler.Registered(bean+".x") || registeredBean(bean)
}

// registeredBean 该 bean 是否在内置任务注册表里（ryTask）。
func registeredBean(bean string) bool {
	for _, n := range []string{"ryNoParams", "ryParams", "ryMultipleParams"} {
		if scheduler.Registered(bean + "." + n) {
			return true
		}
	}
	return false
}

func stringOrDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// ---------- 调度日志：对应----------

// SelectJobLogList 分页查询，等价。
func SelectJobLogList(q *query.JobLogList, p *page.PageDomain) ([]entity.SysJobLog, int64, error) {
	return repository.SelectJobLogList(q, p)
}

// SelectJobLogByID 详情，等价。
func SelectJobLogByID(id int64) (*entity.SysJobLog, error) {
	return repository.SelectJobLogByID(id)
}

// DeleteJobLogByIDs 批量删除，等价。
func DeleteJobLogByIDs(ids []int64) (int64, error) {
	return repository.DeleteJobLogByIDs(ids)
}

// CleanJobLog 清空，等价。
func CleanJobLog() error {
	return repository.CleanJobLog()
}

// ---------- 导出 ----------

// JobExportRows 定时任务导出的表头与数据行（列名取自 SysJob 的 @Excel）。
func JobExportRows(rows []entity.SysJob) ([]string, [][]any) {
	headers := []string{"任务序号", "任务名称", "任务组名", "调用目标字符串", "执行表达式 ", "计划策略 ", "并发执行", "任务状态"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.JobID, r.JobName, r.JobGroup, r.InvokeTarget,
			r.CronExpression, misfireName(r.MisfirePolicy), concurrentName(r.Concurrent), statusName(r.Status)})
	}
	return headers, out
}

// JobLogExportRows 调度日志导出的表头与数据行（列名取自 SysJobLog 的 @Excel）。
func JobLogExportRows(rows []entity.SysJobLog) ([]string, [][]any) {
	headers := []string{"日志序号", "任务名称", "任务组名", "调用目标字符串", "日志信息", "执行状态", "异常信息"}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		st := "正常"
		if r.Status == "1" {
			st = "失败"
		}
		out = append(out, []any{r.JobLogID, r.JobName, r.JobGroup, r.InvokeTarget,
			r.JobMessage, st, r.ExceptionInfo})
	}
	return headers, out
}

// 下面三个字典翻译对应 @Excel 的 readConverterExp。
func misfireName(v string) string {
	switch v {
	case "0":
		return "默认"
	case "1":
		return "立即触发执行"
	case "2":
		return "触发一次执行"
	case "3":
		return "不触发立即执行"
	}
	return ""
}

func concurrentName(v string) string {
	switch v {
	case "0":
		return "允许"
	case "1":
		return "禁止"
	}
	return ""
}

func statusName(v string) string {
	switch v {
	case "0":
		return "正常"
	case "1":
		return "暂停"
	}
	return ""
}
