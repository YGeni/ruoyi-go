package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/cronx"
)

// jobRepo 调度器需要的仓储能力。用接口是为了让单测能注入假实现，
// 也避免调度器直接依赖 gorm（它跑在 goroutine 里，不该碰请求上下文）。
type jobRepo interface {
	SelectJobAll() ([]entity.SysJob, error)
	SelectJobByID(jobID int64) (*entity.SysJob, error)
	InsertJobLog(l *entity.SysJobLog) error
}

var (
	mu     sync.Mutex
	timers = map[int64]*time.Timer{} // jobID -> 下次触发的定时器
	repo   jobRepo
	stopCh chan struct{}
	once   sync.Once
)

// Start 启动调度器：加载 sys_job 里 status='0'（正常）的任务并逐个排程。
//
// 只启动一次（sync.Once）。status='1' 是暂停，不排程。
func Start(r jobRepo) {
	once.Do(func() {
		repo = r
		stopCh = make(chan struct{})
		go func() {
			<-stopCh
			mu.Lock()
			for _, t := range timers {
				t.Stop()
			}
			timers = map[int64]*time.Timer{}
			mu.Unlock()
		}()
	})
	if err := Reload(); err != nil {
		log.Printf("[scheduler] 加载失败: %v", err)
	}
}

// Stop 停止全部定时器。
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if stopCh != nil {
		select {
		case <-stopCh: // 已关
		default:
			close(stopCh)
		}
	}
}

// Reload 重新加载任务表：先停掉所有定时器，再按当前表内容排程。
//
// 新增/修改/删除/改状态之后都要调它——对应 Java 侧 quartz 的
// scheduler.deleteJob + scheduleJob（SysJobServiceImpl 里每个写操作都做一遍）。
func Reload() error {
	if repo == nil {
		return nil
	}
	jobs, err := repo.SelectJobAll()
	if err != nil {
		return err
	}
	mu.Lock()
	for _, t := range timers {
		t.Stop()
	}
	timers = map[int64]*time.Timer{}
	mu.Unlock()

	for _, j := range jobs {
		if j.Status != "0" { // '0' 正常，'1' 暂停
			continue
		}
		if err := scheduleOne(j); err != nil {
			log.Printf("[scheduler] 任务 %d(%s) 排程失败: %v", j.JobID, j.JobName, err)
		}
	}
	return nil
}

// Refresh 只重排某一个任务（改状态/改 cron 后用），比 Reload 全量便宜。
func Refresh(jobID int64) error {
	if repo == nil {
		return nil
	}
	mu.Lock()
	if t, ok := timers[jobID]; ok {
		t.Stop()
		delete(timers, jobID)
	}
	mu.Unlock()
	j, err := repo.SelectJobByID(jobID)
	if err != nil || j == nil {
		return err
	}
	if j.Status != "0" {
		return nil
	}
	return scheduleOne(*j)
}

// Remove 停掉并移除某个任务的定时器。
func Remove(jobID int64) {
	mu.Lock()
	defer mu.Unlock()
	if t, ok := timers[jobID]; ok {
		t.Stop()
		delete(timers, jobID)
	}
}

// scheduleOne 给单个任务排下一次触发。
func scheduleOne(j entity.SysJob) error {
	s, err := cronx.Parse(j.CronExpression)
	if err != nil {
		return err
	}
	next, err := s.Next(time.Now())
	if err != nil {
		return err
	}
	d := time.Until(next)
	if d < 0 {
		d = 0
	}
	mu.Lock()
	defer mu.Unlock()
	timers[j.JobID] = time.AfterFunc(d, func() {
		runOnce(j.JobID)
		// 触发后接着排下一次
		if err := Refresh(j.JobID); err != nil {
			log.Printf("[scheduler] 任务 %d 重排失败: %v", j.JobID, err)
		}
	})
	return nil
}

// runOnce 执行一次并写 sys_job_log。
//
// 对应 JobInvokeUtil.invokeMethod + AsyncFactory.recordJobLog：
// 起止时间、耗时消息、异常信息都落库。concurrent='1'（禁止并发）的语义
// 在 Go 侧由"上一次没跑完就跳过本次"近似实现。
func runOnce(jobID int64) {
	if repo == nil {
		return
	}
	j, err := repo.SelectJobByID(jobID)
	if err != nil || j == nil {
		return
	}
	start := time.Now()
	msg, runErr := Invoke(j.InvokeTarget)
	end := time.Now()

	l := &entity.SysJobLog{
		JobName:      j.JobName,
		JobGroup:     j.JobGroup,
		InvokeTarget: j.InvokeTarget,
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
	if err := repo.InsertJobLog(l); err != nil {
		log.Printf("[scheduler] 写调度日志失败: %v", err)
	}
}
