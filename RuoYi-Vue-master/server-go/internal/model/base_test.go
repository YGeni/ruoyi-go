package model_test

import (
	"testing"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model"
)

// TestBaseModelBeforeCreate 覆盖 create_time / update_time 的填充。
//
// 背景（AGENTS.md §12 风险 3）：原库所有表的 create_time / update_time
// **均无数据库默认值**，Java 侧由各 Mapper 的 sysdate() 写入；
// Go 侧改为 BaseModel 的 hook 填充，因此必须保证：
//   - 新建时两个时间都被填上
//   - 已自带 create_time 时不被覆盖
func TestBaseModelBeforeCreate(t *testing.T) {
	cases := []struct {
		name       string
		preset     *time.Time
		wantPreset bool
	}{
		{name: "未设置时间时自动填充", preset: nil, wantPreset: false},
		{name: "已设置 create_time 时保持不变", preset: timePtr(time.Date(2026, 4, 17, 10, 0, 0, 0, time.Local)), wantPreset: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &model.BaseModel{}
			if tc.preset != nil {
				b.CreateTime = tc.preset
			}
			if err := b.BeforeCreate(nil); err != nil {
				t.Fatalf("BeforeCreate 返回错误: %v", err)
			}
			if b.CreateTime == nil {
				t.Fatal("create_time 未填充")
			}
			if b.UpdateTime == nil {
				t.Fatal("update_time 未填充")
			}
			if tc.wantPreset && !b.CreateTime.Equal(*tc.preset) {
				t.Fatalf("create_time 被覆盖：期望 %v，实际 %v", *tc.preset, *b.CreateTime)
			}
		})
	}
}

// TestBaseModelBeforeUpdate 覆盖：更新只刷新 update_time，绝不动 create_time。
func TestBaseModelBeforeUpdate(t *testing.T) {
	created := time.Date(2026, 1, 1, 8, 0, 0, 0, time.Local)
	b := &model.BaseModel{CreateTime: &created}
	// 先造一个旧的 update_time，确保会被刷新
	old := time.Date(2026, 1, 2, 8, 0, 0, 0, time.Local)
	b.UpdateTime = &old

	if err := b.BeforeUpdate(nil); err != nil {
		t.Fatalf("BeforeUpdate 返回错误: %v", err)
	}
	if !b.CreateTime.Equal(created) {
		t.Fatalf("create_time 不应被修改：期望 %v，实际 %v", created, *b.CreateTime)
	}
	if !b.UpdateTime.After(old) {
		t.Fatalf("update_time 应被刷新为更晚的时间：期望晚于 %v，实际 %v", old, *b.UpdateTime)
	}
}

// TestDelFlagConstants 覆盖 del_flag 三态字符串语义（'0' 存在 / '2' 删除）。
//
// 背景（AGENTS.md §12 风险 3）：禁止使用 gorm.DeletedAt，因为该列是 char(1)
// 且软删值为 '2'，与 DeletedAt 的 NULL 语义相反。
func TestDelFlagConstants(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		wantDel bool
	}{
		{name: "0 表示存在", flag: "0", wantDel: false},
		{name: "2 表示已删除", flag: "2", wantDel: true},
		{name: "1 停用不等于删除", flag: "1", wantDel: false},
		{name: "空值不等于删除", flag: "", wantDel: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := model.IsDeleted(tc.flag); got != tc.wantDel {
				t.Fatalf("IsDeleted(%q) 期望 %v，实际 %v", tc.flag, tc.wantDel, got)
			}
		})
	}
	if model.DelFlagExists != "0" || model.DelFlagDeleted != "2" {
		t.Fatalf("del_flag 常量取值错误：exists=%q deleted=%q", model.DelFlagExists, model.DelFlagDeleted)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
