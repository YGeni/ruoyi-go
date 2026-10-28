package repository

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"gorm.io/gorm"
)

// ---------- 登录日志 sys_logininfor：对应 SysLogininforMapper.xml ----------

// SelectLogininforList 分页查询登录日志，等价 SysLogininforMapper.xml:26-45。
//
// 无 DataScope：SysLogininforController.java:40 的 selectLogininforList 不带
// @DataScope 注解，所有登录记录对有 monitor:logininfor:list 权限的人可见。
func SelectLogininforList(q *query.LogininforList, p *page.PageDomain) ([]entity.SysLogininfor, int64, error) {
	tx := db.DB.Table("sys_logininfor").Order("info_id desc")
	if q.Ipaddr != "" {
		tx = tx.Where("ipaddr LIKE ?", "%"+q.Ipaddr+"%")
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.UserName != "" {
		tx = tx.Where("user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.BeginTime != "" {
		tx = tx.Where("login_time >= ?", q.BeginTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("login_time <= ?", q.EndTime)
	}
	var rows []entity.SysLogininfor
	total, err := page.Paginate(tx, p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// DeleteLogininforByIDs 批量删除，等价 SysLogininforMapper.xml deleteLogininforByIds。
func DeleteLogininforByIDs(infoIDs []int64) (int64, error) {
	if len(infoIDs) == 0 {
		return 0, nil
	}
	tx := db.DB.Where("info_id IN ?", infoIDs).Delete(&entity.SysLogininfor{})
	return tx.RowsAffected, tx.Error
}

// CleanLogininfor 清空表，等价 SysLogininforMapper.xml cleanLogininfor（truncate）。
func CleanLogininfor() error {
	return db.DB.Exec("TRUNCATE TABLE sys_logininfor").Error
}

// ---------- 操作日志 sys_oper_log：对应 SysOperLogMapper.xml ----------

// SelectOperLogList 分页查询操作日志，等价 SysOperLogMapper.xml selectOperLogList。
//
// 无 DataScope（SysOperlogController.java:36 的 selectOperLogList 不带 @DataScope）。
func SelectOperLogList(q *query.OperLogList, p *page.PageDomain) ([]entity.SysOperLog, int64, error) {
	tx := db.DB.Table("sys_oper_log").Order("oper_id desc")
	if q.OperIp != "" {
		tx = tx.Where("oper_ip LIKE ?", "%"+q.OperIp+"%")
	}
	if q.Title != "" {
		tx = tx.Where("title LIKE ?", "%"+q.Title+"%")
	}
	if q.BusinessType != nil {
		tx = tx.Where("business_type = ?", *q.BusinessType)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.OperName != "" {
		tx = tx.Where("oper_name LIKE ?", "%"+q.OperName+"%")
	}
	if q.BeginTime != "" {
		tx = tx.Where("oper_time >= ?", q.BeginTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("oper_time <= ?", q.EndTime)
	}
	var rows []entity.SysOperLog
	total, err := page.Paginate(tx, p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// DeleteOperLogByIDs 批量删除，等价 SysOperLogMapper.xml deleteOperLogByIds。
func DeleteOperLogByIDs(operIDs []int64) (int64, error) {
	if len(operIDs) == 0 {
		return 0, nil
	}
	tx := db.DB.Where("oper_id IN ?", operIDs).Delete(&entity.SysOperLog{})
	return tx.RowsAffected, tx.Error
}

// CleanOperLog 清空表，等价 SysOperLogMapper.xml cleanOperLog（truncate）。
func CleanOperLog() error {
	return db.DB.Exec("TRUNCATE TABLE sys_oper_log").Error
}

// SelectOperLogByID 单条查询。
func SelectOperLogByID(operID int64) (*entity.SysOperLog, error) {
	var row entity.SysOperLog
	err := db.DB.Where("oper_id = ?", operID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
