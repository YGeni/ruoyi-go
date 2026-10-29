package service

import (
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// noticeTopLimit 首页顶部公告条数上限，
// 与的硬编码 selectNoticeListWithReadStatus(userId, 5) 一致。
const noticeTopLimit = 5

// SelectNoticeList 公告列表，等价 SysNoticeServiceImpl.selectNoticeList。
//
// 公告**没有** @DataScope 注解（不在 DataScope 五个落点内）。
func SelectNoticeList(q *query.NoticeList, p *page.PageDomain) ([]entity.SysNotice, int64, error) {
	return repository.SelectNoticeList(q, p)
}

// SelectNoticeByID 按主键查公告。
func SelectNoticeByID(noticeID int64) (*entity.SysNotice, error) {
	return repository.SelectNoticeByID(noticeID)
}

// InsertNotice 新增公告，等价 SysNoticeServiceImpl.insertNotice。
func InsertNotice(x *entity.SysNotice) error {
	return repository.InsertNotice(x)
}

// UpdateNotice 修改公告，等价 SysNoticeServiceImpl.updateNotice。
func UpdateNotice(x *entity.SysNotice) error {
	return repository.UpdateNotice(x)
}

// DeleteNoticeByIDs 删除公告，等价 SysNoticeServiceImpl.deleteNoticeByIds。
//
// 同时清理 sys_notice_read（见 repository.DeleteNoticeByIDs 的 DIFF 说明）。
func DeleteNoticeByIDs(ids []int64) error {
	return repository.DeleteNoticeByIDs(ids)
}

// ---------------------------------------------------------------- 3.9 新增端点

// SelectNoticeListTop 首页顶部公告（带已读标记），
// 等价的 listTop。
//
// Java 在 controller 里用 stream().filter(n -> !n.getIsRead()).count() 算未读数，
// 再 result.put("unreadCount", unreadCount)；Go 侧在 service 里算好一并返回，
// handler 只需组装 AjaxResult{data, unreadCount}。
func SelectNoticeListTop(userID int64) ([]entity.SysNoticeWithRead, int64, error) {
	rows, err := repository.SelectNoticeListWithReadStatus(userID, noticeTopLimit)
	if err != nil {
		return nil, 0, err
	}
	var unread int64
	for i := range rows {
		if !rows[i].IsRead {
			unread++
		}
	}
	return rows, unread, nil
}

// MarkNoticeRead 标记单条公告已读，
// 等价的 markRead。
//
// 幂等：sys_notice_read 有 uk_user_notice 唯一约束，重复标记不报错也不产生重复行。
func MarkNoticeRead(noticeID, userID int64) error {
	if noticeID == 0 {
		return nil
	}
	return repository.InsertNoticeRead(noticeID, userID)
}

// MarkNoticeReadAll 批量标记已读，
// 等价的 markReadAll（ids 为逗号分隔字符串）。
func MarkNoticeReadAll(userID int64, noticeIDs []int64) error {
	return repository.MarkReadBatch(userID, noticeIDs)
}

// SelectNoticeReadUsers 已读用户列表（/system/notice/readUsers/list），
// 等价+ SysNoticeReadMapper.xml:73-95。
func SelectNoticeReadUsers(q *query.ReadUserList, p *page.PageDomain) ([]entity.NoticeReadUser, int64, error) {
	return repository.SelectReadUsersByNoticeID(q.NoticeID, q.SearchValue, p)
}

// CountNoticeUnread 未读公告数，等价 SysNoticeReadMapper.xml selectUnreadCount。
func CountNoticeUnread(userID int64) (int64, error) {
	return repository.SelectUnreadCount(userID)
}
