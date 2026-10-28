package repository

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"gorm.io/gorm"
)

// ---------- sys_job ----------

// SelectJobList 分页查询，等价 SysJobServiceImpl.java:57-61。
func SelectJobList(q *query.JobList, p *page.PageDomain) ([]entity.SysJob, int64, error) {
	qb := db.DB.Model(&entity.SysJob{})
	if q != nil {
		if q.JobName != "" {
			qb = qb.Where("job_name like ?", "%"+q.JobName+"%")
		}
		if q.JobGroup != "" {
			qb = qb.Where("job_group = ?", q.JobGroup)
		}
		if q.Status != "" {
			qb = qb.Where("status = ?", q.Status)
		}
		if q.InvokeTarget != "" {
			qb = qb.Where("invoke_target like ?", "%"+q.InvokeTarget+"%")
		}
	}
	var rows []entity.SysJob
	total, err := page.Paginate(qb, p, "", "*", &rows)
	return rows, total, err
}

// SelectJobByID 按主键查，等价 SysJobServiceImpl.java:69-73。
func SelectJobByID(jobID int64) (*entity.SysJob, error) {
	var j entity.SysJob
	err := db.DB.Where("job_id = ?", jobID).First(&j).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// SelectJobAll 全量查询，给调度器加载用（SysJobServiceImpl.java:161-165 的
// selectJobAll 同样是不分页的）。
func SelectJobAll() ([]entity.SysJob, error) {
	var rows []entity.SysJob
	err := db.DB.Find(&rows).Error
	return rows, err
}

// InsertJob 新增，等价 SysJobServiceImpl.java:81-93。
func InsertJob(j *entity.SysJob) error {
	now := time.Now()
	j.CreateTime = &now
	return db.DB.Create(j).Error
}

// UpdateJob 修改，等价 SysJobServiceImpl.java:100-113。
func UpdateJob(j *entity.SysJob) error {
	now := time.Now()
	j.UpdateTime = &now
	return db.DB.Where("job_id = ?", j.JobID).
		Select("job_name", "job_group", "invoke_target", "cron_expression",
			"misfire_policy", "concurrent", "status", "remark", "update_by", "update_time").
		Updates(j).Error
}

// UpdateJobStatus 只改状态，等价 SysJobServiceImpl.java:121-131。
func UpdateJobStatus(j *entity.SysJob) error {
	now := time.Now()
	j.UpdateTime = &now
	return db.DB.Model(&entity.SysJob{}).Where("job_id = ?", j.JobID).
		Updates(map[string]any{"status": j.Status, "update_by": j.UpdateBy, "update_time": now}).Error
}

// DeleteJobByIDs 批量删除，等价 SysJobServiceImpl.java:139-155。
func DeleteJobByIDs(jobIDs []int64) (int64, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}
	tx := db.DB.Where("job_id IN ?", jobIDs).Delete(&entity.SysJob{})
	return tx.RowsAffected, tx.Error
}

// ---------- sys_job_log ----------

// SelectJobLogList 分页查询，等价 SysJobLogServiceImpl.java:44-48。
func SelectJobLogList(q *query.JobLogList, p *page.PageDomain) ([]entity.SysJobLog, int64, error) {
	qb := db.DB.Model(&entity.SysJobLog{})
	if q != nil {
		if q.JobName != "" {
			qb = qb.Where("job_name like ?", "%"+q.JobName+"%")
		}
		if q.JobGroup != "" {
			qb = qb.Where("job_group = ?", q.JobGroup)
		}
		if q.Status != "" {
			qb = qb.Where("status = ?", q.Status)
		}
		if q.InvokeTarget != "" {
			qb = qb.Where("invoke_target like ?", "%"+q.InvokeTarget+"%")
		}
		if q.BeginTime != "" {
			qb = qb.Where("create_time >= ?", q.BeginTime)
		}
		if q.EndTime != "" {
			qb = qb.Where("create_time <= ?", q.EndTime)
		}
	}
	var rows []entity.SysJobLog
	total, err := page.Paginate(qb, p, "", "*", &rows)
	return rows, total, err
}

// SelectJobLogByID 按主键查，等价 SysJobLogServiceImpl.java:56-60。
func SelectJobLogByID(id int64) (*entity.SysJobLog, error) {
	var l entity.SysJobLog
	err := db.DB.Where("job_log_id = ?", id).First(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// InsertJobLog 写调度日志，等价 AsyncFactory.recordJobLog 的落库部分。
func InsertJobLog(l *entity.SysJobLog) error {
	return db.DB.Create(l).Error
}

// DeleteJobLogByIDs 批量删除，等价 SysJobLogServiceImpl.java:78-82。
func DeleteJobLogByIDs(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tx := db.DB.Where("job_log_id IN ?", ids).Delete(&entity.SysJobLog{})
	return tx.RowsAffected, tx.Error
}

// CleanJobLog 清空，等价 SysJobLogServiceImpl.java:90-93 的
// jobLogMapper.cleanJobLog()（truncate 语义，用无条件删除实现）。
func CleanJobLog() error {
	return db.DB.Where("1 = 1").Delete(&entity.SysJobLog{}).Error
}
