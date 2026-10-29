package excelx

import (
	"bytes"
	"testing"
)

// TestWriteReadRoundtrip 导出再读回来，列名与数据必须一致。
//
// 导入功能依赖 Read，导出依赖 Write：两边对不上就会出现
// "自己导出的文件自己导入不了"。
func TestWriteReadRoundtrip(t *testing.T) {
	headers := []string{"登录名称", "用户名称", "用户性别", "帐号状态"}
	rows := [][]any{{"alice", "爱丽丝", "女", "正常"}, {"bob", "鲍勃", "男", "停用"}}
	var buf bytes.Buffer
	if err := Write(&buf, "用户数据", headers, rows); err != nil {
		t.Fatalf("Write 失败: %v", err)
	}
	got, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read 失败: %v", err)
	}
	if len(got) != len(rows)+1 {
		t.Fatalf("行数应为 %d（表头+数据），实际 %d: %v", len(rows)+1, len(got), got)
	}
	for i, h := range headers {
		if got[0][i] != h {
			t.Errorf("第 %d 列表头 = %q，应为 %q", i, got[0][i], h)
		}
	}
	for r, row := range rows {
		for c, want := range row {
			if got[r+1][c] != want {
				t.Errorf("第 %d 行第 %d 列 = %q，应为 %q", r+1, c, got[r+1][c], want)
			}
		}
	}
}
