// Package excelx 导入导出 xlsx，等价 ruoyi-common 的 ExcelUtil
// （exportExcel / importExcel 两个方向）。
//
// Java 侧是 POI：导出直接写 HttpServletResponse，导入读 MultipartFile 的流。
// Go 侧用 excelize 生成/解析字节流。
//
// 前端 download（request.js:127-151）以 responseType: 'blob' 接收，
// blobValidate（ruoyi.js:226-228）的判定是 **type !== 'application/json'**，
// 所以只要不返回 JSON 就会走 saveAs 落盘。
package excelx

import (
	"errors"
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

// Read 读第一个工作表的全部行，等价 ExcelUtil.importExcel 的读文件部分。
//
// 返回的每行按**列**对齐：excelize 的 GetRows 会跳过行尾空单元格，
// 所以短行要按调用方期望的列数自行补空（导入时按表头长度补）。
func Read(r io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("Excel 里没有工作表")
	}
	return f.GetRows(sheets[0])
}
