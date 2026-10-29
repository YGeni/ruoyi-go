package system

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// JobHandler 定时任务 + 调度日志，对应与
// （都在 ruoyi-quartz 模块下）。
type JobHandler struct{}

// ---------- 定时任务：对应----------

// JobList GET /monitor/job/list
//
// 等价：startPage + getDataTable。
// 前端 src/views/monitor/job/index.vue:285-286 读 response.rows / response.total。
func (JobHandler) JobList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.JobList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectJobList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysJob{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// JobExport POST /monitor/job/export
//
// 与登录日志/操作日志同理：download 走 form-urlencoded，用 ShouldBind 而非 Query。
func (JobHandler) JobExport(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.JobList
	if err := c.ShouldBind(&q); err != nil {
		fail(c, err)
		return
	}
	rows, _, err := service.SelectJobList(&q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	headers, data := service.JobExportRows(rows)
	writeXlsx(c, "定时任务", headers, data)
}

// JobGet GET /monitor/job/:jobId
func (JobHandler) JobGet(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, ok := pathID(c, "jobId")
	if !ok {
		return
	}
	j, err := service.SelectJobByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(j))
}

// JobAdd POST /monitor/job
func (JobHandler) JobAdd(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var j entity.SysJob
	if err := c.ShouldBindJSON(&j); err != nil {
		fail(c, err)
		return
	}
	lu, _ := loginUser(c)
	if lu != nil {
		j.CreateBy = lu.Username()
	}
	if err := service.InsertJob(&j); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// JobEdit PUT /monitor/job
func (JobHandler) JobEdit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var j entity.SysJob
	if err := c.ShouldBindJSON(&j); err != nil {
		fail(c, err)
		return
	}
	lu, _ := loginUser(c)
	if lu != nil {
		j.UpdateBy = lu.Username()
	}
	if err := service.UpdateJob(&j); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// JobChangeStatus PUT /monitor/job/changeStatus
//
// 等价：先查库再改状态（Java 用 selectJobById
// 拿到 newJob 再 setStatus，避免前端少传字段把别的列冲掉）。
func (JobHandler) JobChangeStatus(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var body struct {
		JobID  int64  `json:"jobId"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, err)
		return
	}
	j := &entity.SysJob{JobID: body.JobID, Status: body.Status}
	lu, _ := loginUser(c)
	if lu != nil {
		j.UpdateBy = lu.Username()
	}
	if err := service.ChangeStatus(j); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// JobRun PUT /monitor/job/run
//
// 等价：成功 success
// 任务不存在时 error("任务不存在或已过期！")。
func (JobHandler) JobRun(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var body struct {
		JobID int64 `json:"jobId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, err)
		return
	}
	if err := service.Run(&entity.SysJob{JobID: body.JobID}); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// JobRemove DELETE /monitor/job/:jobIds
func (JobHandler) JobRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.DeleteJobByIDs(parseIDs(c.Param("jobIds"))); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// ---------- 调度日志：对应----------

// JobLogList GET /monitor/jobLog/list
//
// 前端 src/views/monitor/job/log.vue:175-176 读 response.rows / response.total。
func (JobHandler) JobLogList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.JobLogList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, total, err := service.SelectJobLogList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysJobLog{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// JobLogExport POST /monitor/jobLog/export
func (JobHandler) JobLogExport(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.JobLogList
	if err := c.ShouldBind(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, _, err := service.SelectJobLogList(&q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	headers, data := service.JobLogExportRows(rows)
	writeXlsx(c, "调度日志", headers, data)
}

// JobLogGet GET /monitor/jobLog/:jobLogId
func (JobHandler) JobLogGet(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, ok := pathID(c, "jobLogId")
	if !ok {
		return
	}
	l, err := service.SelectJobLogByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(l))
}

// JobLogRemove DELETE /monitor/jobLog/:jobLogIds
func (JobHandler) JobLogRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	n, err := service.DeleteJobLogByIDs(parseIDs(c.Param("jobLogIds")))
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.ToAjaxRows(int(n)))
}

// JobLogClean DELETE /monitor/jobLog/clean
func (JobHandler) JobLogClean(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.CleanJobLog(); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// pathID 解析路径里的单个 id。
//
// Java 的 @PathVariable Long jobId 遇到非数字会抛
// MethodArgumentTypeMismatchException（400），Go 侧回 500 + "操作失败"。
func pathID(c *gin.Context, name string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil {
		result.Fail(c, result.Error(config.MsgOperateFailed))
		return 0, false
	}
	return v, true
}
