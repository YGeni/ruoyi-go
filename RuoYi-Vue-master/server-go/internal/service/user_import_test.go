package service

// 导入的解析逻辑：列顺序 + 字典翻译（男/女、正常/停用）。
//
// 导出写的是中文，导入要能读回自己导出的文件 —— parseSex/parseStatus
// 两边必须对称，否则"导出再导入"会静默把性别/状态改掉。

import (
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
)

func TestImportUserRow(t *testing.T) {
	cases := []struct {
		name    string
		row     []string
		want    entity.SysUser
		wantErr bool
	}{
		{"中文性别状态", []string{"alice", "爱丽丝", "a@b.c", "13800000000", "女", "停用"},
			entity.SysUser{UserName: "alice", NickName: "爱丽丝", Email: "a@b.c", Phonenumber: "13800000000", Sex: "1", Status: config.Disable}, false},
		{"原始字典值", []string{"bob", "鲍勃", "", "", "0", "1"},
			entity.SysUser{UserName: "bob", NickName: "鲍勃", Sex: "0", Status: config.Disable}, false},
		{"未知性别默认未知", []string{"carol", "卡罗", "", "", "", ""},
			entity.SysUser{UserName: "carol", NickName: "卡罗", Sex: "2", Status: config.Normal}, false},
		{"昵称为空回落登录名", []string{"dave", "", "", "", "", ""},
			entity.SysUser{UserName: "dave", NickName: "dave", Sex: "2", Status: config.Normal}, false},
		{"登录名为空要报错", []string{"", "无名", "", "", "", ""}, entity.SysUser{}, true},
		{"列数不足要报错", []string{"eve"}, entity.SysUser{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := importUserRow(c.row)
			if c.wantErr {
				if err == nil {
					t.Fatalf("应报错但没有")
				}
				return
			}
			if err != nil {
				t.Fatalf("不该报错: %v", err)
			}
			if got.UserName != c.want.UserName || got.NickName != c.want.NickName ||
				got.Email != c.want.Email || got.Phonenumber != c.want.Phonenumber ||
				got.Sex != c.want.Sex || got.Status != c.want.Status {
				t.Errorf("解析结果 = %+v，应为 %+v", *got, c.want)
			}
		})
	}
}

// TestExportImportSexSymmetric 导出翻译与导入解析必须互为逆运算。
func TestExportImportSexSymmetric(t *testing.T) {
	for _, s := range []string{"0", "1", "2"} {
		u, err := importUserRow([]string{"u" + s, "n", "", "", sexText(s), statusText(config.Normal)})
		if err != nil {
			t.Fatalf("导入失败: %v", err)
		}
		if u.Sex != s {
			t.Errorf("sex %s 导出成 %q 再导入变成 %s", s, sexText(s), u.Sex)
		}
	}
}
