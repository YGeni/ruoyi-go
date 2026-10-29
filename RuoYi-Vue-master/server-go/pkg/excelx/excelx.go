// Package excelx 导出 xlsx，等价 ruoyi-common 的 ExcelUtil.exportExcel。
//
// Java 侧是 POI 直接写 HttpServletResponse
// Go 侧用 excelize 生成字节流，由 handler 设好响应头后写回。
//
// 前端 download（request.js:127-151）以 responseType: 'blob' 接收，
// blobValidate（ruoyi.js:226-228）的判定是 **type !== 'application/json'**，
// 所以只要不返回 JSON 就会走 saveAs 落盘。
package excelx

import (
	"io"

	"github.com/xuri/excelize/v2"
)

// ContentType xlsx 的 MIME。
const ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// DefaultSheetName 默认工作表名。
const DefaultSheetName = "Sheet1"

// Write 把表头 + 数据行写成 xlsx 写入 w。
//
// 列名取自各 domain 上的 @Excel(name = "...") 注解，顺序与 Java 导出一致。
func Write(w io.Writer, sheet string, headers []string, rows [][]any) error {
	if sheet == "" {
		sheet = DefaultSheetName
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// excelize 新建的工作簿自带 "Sheet1"；换名字后要把默认的删掉，
	// 否则导出文件里会多一个空表。
	if sheet == DefaultSheetName {
		if err := f.SetSheetName(DefaultSheetName, sheet); err != nil {
			return err
		}
	} else {
		idx, err := f.NewSheet(sheet)
		if err != nil {
			return err
		}
		f.SetActiveSheet(idx)
		if err := f.DeleteSheet(DefaultSheetName); err != nil {
			return err
		}
	}

	if len(headers) > 0 {
		cells := make([]any, len(headers))
		for i, h := range headers {
			cells[i] = h
		}
		if err := f.SetSheetRow(sheet, "A1", &cells); err != nil {
			return err
		}
	}
	for i, r := range rows {
		cells := make([]any, len(r))
		copy(cells, r)
		cell, err := excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return err
		}
		if err := f.SetSheetRow(sheet, cell, &cells); err != nil {
			return err
		}
	}
	return f.Write(w)
}
