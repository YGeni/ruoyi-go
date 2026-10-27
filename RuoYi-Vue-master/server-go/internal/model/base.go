// Package model 定义公共模型基类与三态删除标记。
package model

import (
	"time"

	"gorm.io/gorm"
)

// 删除标记（del_flag，char(1) 三态字符串）。
//
// 来源：sql/ry_20260417.sql 中 sys_user/sys_role/sys_dept 的 del_flag char(1) default '0'，
// 取值 '0' 存在 / '2' 删除（见 UserStatus: OK("0") / DISABLE("1") / DELETED("2")）。
//
// DIFF(review): 不使用 gorm.DeletedAt。原因有二：
//  1. del_flag 是 char(1) 且软删值为 '2' 而非时间戳/布尔，DeletedAt 的语义（NULL 未删）
//     与既有数据完全相反，直接套用会查不到任何历史数据；
//  2. 31 张表是 Java/Go 两端的共同契约（AGENTS.md 全局规则 2），禁止改动表结构。
const (
	DelFlagExists  = "0"
	DelFlagDeleted = "2"
)

// IsDeleted 判断删除标记是否为"已删除"。
func IsDeleted(flag string) bool { return flag == DelFlagDeleted }

// BaseModel 对应 Java 的 BaseEntity（create_by/create_time/update_by/update_time/remark）。
//
// DIFF(review): 原库所有表的 create_time/update_time 均**无数据库默认值**
// （sql/ 内无 CURRENT_TIMESTAMP / ON UPDATE CURRENT_TIMESTAMP），Java 侧由各
// Mapper 的 sysdate() 写入。Go 侧统一由这两个 hook 显式填充，避免漏写导致零值时间。
type BaseModel struct {
	CreateBy   string     `gorm:"column:create_by" json:"createBy,omitempty"`
	CreateTime *time.Time `gorm:"column:create_time" json:"createTime,omitempty"`
	UpdateBy   string     `gorm:"column:update_by" json:"updateBy,omitempty"`
	UpdateTime *time.Time `gorm:"column:update_time" json:"updateTime,omitempty"`
	Remark     string     `gorm:"column:remark" json:"remark,omitempty"`
}

// BeforeCreate 新建时填充 create_time / update_time。
func (b *BaseModel) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if b.CreateTime == nil {
		b.CreateTime = &now
	}
	if b.UpdateTime == nil {
		b.UpdateTime = &now
	}
	return nil
}

// BeforeUpdate 更新时只刷新 update_time，绝不覆盖 create_time。
func (b *BaseModel) BeforeUpdate(_ *gorm.DB) error {
	now := time.Now()
	b.UpdateTime = &now
	return nil
}
