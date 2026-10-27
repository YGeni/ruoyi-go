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

// ---------------------------------------------------------------- 公告主体

// SelectNoticeList 分页查询公告，等价 SysNoticeMapper.xml:29-42。
//
// Java：
//
//	select ... from sys_notice
//	where 1=1 [and notice_title like ?] [and notice_type=?] [and create_by like ?]
//	order by notice_id desc
//
// 公告**没有** @DataScope 注解（SysNoticeServiceImpl.selectNoticeList 无数据权限）。
func SelectNoticeList(q *query.NoticeList, p *page.PageDomain) ([]entity.SysNotice, int64, error) {
	tx := db.DB.Table("sys_notice").Order("notice_id desc")
	if q.NoticeTitle != "" {
		tx = tx.Where("notice_title LIKE ?", "%"+q.NoticeTitle+"%")
	}
	if q.NoticeType != "" {
		tx = tx.Where("notice_type = ?", q.NoticeType)
	}
	if q.CreateBy != "" {
		tx = tx.Where("create_by LIKE ?", "%"+q.CreateBy+"%")
	}
	var rows []entity.SysNotice
	total, err := page.Paginate(tx, p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// SelectNoticeByID 按主键查公告，等价 SysNoticeMapper.xml selectNoticeById。
func SelectNoticeByID(noticeID int64) (*entity.SysNotice, error) {
	var x entity.SysNotice
	err := db.DB.Where("notice_id = ?", noticeID).First(&x).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// InsertNotice 新增公告，等价 SysNoticeMapper.xml insertNotice（动态列）。
func InsertNotice(x *entity.SysNotice) error {
	return db.DB.Create(x).Error
}

// UpdateNotice 修改公告，等价 SysNoticeMapper.xml updateNotice 的 <set>。
//
// 注意 notice_content 的 if 条件是 `noticeContent != null`（不含空串判断），
// 因此允许把正文清空 —— 这里用指针无法区分，改为"传入即更新"，
// 由 service 层保证只有真正传了该字段时才赋值。
func UpdateNotice(x *entity.SysNotice) error {
	sets := map[string]any{"update_time": time.Now()}
	if x.NoticeTitle != "" {
		sets["notice_title"] = x.NoticeTitle
	}
	if x.NoticeType != "" {
		sets["notice_type"] = x.NoticeType
	}
	if x.NoticeContent != "" {
		sets["notice_content"] = x.NoticeContent
	}
	if x.Status != "" {
		sets["status"] = x.Status
	}
	if x.Remark != "" {
		sets["remark"] = x.Remark
	}
	if x.UpdateBy != "" {
		sets["update_by"] = x.UpdateBy
	}
	return db.DB.Table("sys_notice").Where("notice_id = ?", x.NoticeID).Updates(sets).Error
}

// DeleteNoticeByIDs **物理删除**公告并清理已读记录，
// 等价 SysNoticeMapper.xml deleteNoticeByIds + SysNoticeReadMapper.xml deleteByNoticeIds。
//
// Java 侧先删 sys_notice_read 再删 sys_notice（SysNoticeServiceImpl.deleteNoticeByIds），
// 顺序保持一致；两个操作在同一事务内。
func DeleteNoticeByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notice_id IN ?", ids).Delete(&entity.SysNoticeRead{}).Error; err != nil {
			return err
		}
		return tx.Where("notice_id IN ?", ids).Delete(&entity.SysNotice{}).Error
	})
}

// ---------------------------------------------------------------- 公告已读（3.9 新增）

// InsertNoticeRead 单条标记已读，等价 SysNoticeReadMapper.xml insertNoticeRead。
//
// insert ignore 依赖 uk_user_notice(user_id, notice_id) 唯一约束，重复标记是幂等的。
func InsertNoticeRead(noticeID, userID int64) error {
	return db.DB.Exec(
		"INSERT IGNORE INTO sys_notice_read (notice_id, user_id, read_time) VALUES (?, ?, ?)",
		noticeID, userID, time.Now()).Error
}

// MarkReadBatch 批量标记已读，等价 SysNoticeReadMapper.xml insertNoticeReadBatch。
func MarkReadBatch(userID int64, noticeIDs []int64) error {
	if len(noticeIDs) == 0 {
		return nil
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range noticeIDs {
			if err := tx.Exec(
				"INSERT IGNORE INTO sys_notice_read (notice_id, user_id, read_time) VALUES (?, ?, ?)",
				id, userID, time.Now()).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SelectUnreadCount 未读数，等价 SysNoticeReadMapper.xml selectUnreadCount：
// 正常状态公告数 减去 当前用户已读数。
func SelectUnreadCount(userID int64) (int64, error) {
	var n int64
	err := db.DB.Raw(
		"SELECT COUNT(*) FROM sys_notice n WHERE n.status = '0' "+
			"AND NOT EXISTS (SELECT 1 FROM sys_notice_read r WHERE r.notice_id = n.notice_id AND r.user_id = ?)",
		userID).Scan(&n).Error
	return n, err
}

// SelectNoticeListWithReadStatus 首页顶部公告（带已读标记），
// 等价 SysNoticeReadMapper.xml:44-52。
//
// Java 在 SQL 里直接 limit #{limit}（首页固定 5 条），Go 侧保持一致。
func SelectNoticeListWithReadStatus(userID int64, limit int) ([]entity.SysNoticeWithRead, error) {
	if limit <= 0 {
		limit = 5
	}
	var rows []entity.SysNoticeWithRead
	err := db.DB.Table("sys_notice n").
		Select("n.notice_id AS notice_id, n.notice_title, n.notice_type, n.status, n.create_by, n.create_time, "+
			"CASE WHEN r.notice_id IS NOT NULL THEN 1 ELSE 0 END AS is_read").
		Joins("LEFT JOIN sys_notice_read r ON r.notice_id = n.notice_id AND r.user_id = ?", userID).
		Where("n.status = ?", "0").
		Order("n.notice_id desc").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []entity.SysNoticeWithRead{}
	}
	return rows, nil
}

// SelectReadUsersByNoticeID 已读用户列表，等价 SysNoticeReadMapper.xml:73-95。
//
// searchValue 命中 user_name 或 nick_name 的模糊匹配。
// 排序固定 r.read_time desc。
func SelectReadUsersByNoticeID(noticeID int64, searchValue string, p *page.PageDomain) ([]entity.NoticeReadUser, int64, error) {
	tx := db.DB.Table("sys_notice_read r").
		// **必须逐列传 []string**：传单个逗号分隔字符串时，gorm 会把它按逗号拆开，
		// 每段再套上主表别名 r，拼出 `r`.`user_name` 这类错误列（实测 Error 1054）。
		Select([]string{"u.user_id", "u.user_name", "u.nick_name", "d.dept_name", "u.phonenumber", "r.read_time"}).
		Joins("INNER JOIN sys_user u ON u.user_id = r.user_id AND u.del_flag = '0'").
		Joins("LEFT JOIN sys_dept d ON d.dept_id = u.dept_id").
		Where("r.notice_id = ?", noticeID).
		Order("r.read_time desc")
	if searchValue != "" {
		v := "%" + searchValue + "%"
		tx = tx.Where("(u.user_name LIKE ? OR u.nick_name LIKE ?)", v, v)
	}
	var rows []entity.NoticeReadUser
	total, err := page.Paginate(tx, p, "r", "r.read_id", &rows)
	if err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []entity.NoticeReadUser{}
	}
	return rows, total, nil
}
