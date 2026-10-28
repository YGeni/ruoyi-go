package repository

import "github.com/YGeni/ruoyi-go/server-go/internal/model/entity"

// JobScheduleRepo 给 internal/scheduler 用的仓储适配器。
//
// 调度器只需要这三个方法（接口见 scheduler.jobRepo），
// 单独给个类型是为了让调度器不依赖整个 repository，也方便单测注入假实现。
type JobScheduleRepo struct{}

// SelectJobAll 全量任务。
func (JobScheduleRepo) SelectJobAll() ([]entity.SysJob, error) { return SelectJobAll() }

// SelectJobByID 按主键查。
func (JobScheduleRepo) SelectJobByID(jobID int64) (*entity.SysJob, error) {
	return SelectJobByID(jobID)
}

// InsertJobLog 写调度日志。
func (JobScheduleRepo) InsertJobLog(l *entity.SysJobLog) error { return InsertJobLog(l) }
