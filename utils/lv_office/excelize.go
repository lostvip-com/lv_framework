package lv_office

import (
	"fmt"

	"github.com/lostvip-com/lv_framework/utils/lv_err"
	"github.com/xuri/excelize/v2"
)

func ReadFile(f *excelize.File, sheetName string, colNames []string) ([]map[string]interface{}, error) {
	listArr := make([]map[string]interface{}, 0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for rowIndex, _ := range rows {
		rowNum := rowIndex + 1
		rowMap := make(map[string]interface{}, 0)
		for _, colName := range colNames {
			cellValue, err := f.GetCellValue(sheetName, fmt.Sprintf("%s%d", colName, rowNum))
			rowMap[colName] = cellValue
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
		}
		listArr = append(listArr, rowMap)
	}
	return listArr, err
}

/**
 * 从0 开始
 * GetXlsColTitle
 * 获取excell列标题
 * @param colIndex 列索引
 * @return 列标题
 */
func GetXlsColTitle(colIndex int) string {
	if colIndex < 0 {
		return ""
	}

	var result string
	colIndex++ // 将 0 映射为 1，1 映射为 2，以此类推
	for colIndex > 0 {
		colIndex--
		remainder := colIndex % 26
		result = string(rune('A'+remainder)) + result
		colIndex = colIndex / 26
	}

	return result
}

// GetXlsCellName 1,1 -> A1
func GetXlsCellName(colNum int, rowNum int) string {
	name, err := excelize.CoordinatesToCellName(colNum, rowNum)
	lv_err.IfErrPanic(err)
	return name
}

// FitSheetWidth 自适应列宽度
func FitSheetWidth(f *excelize.File, sheetName string) {
	cols, _ := f.GetCols(sheetName)
	for idx, col := range cols {
		maxLen := 0
		for _, cellValue := range col {
			if len(cellValue) > maxLen {
				maxLen = len(cellValue)
			}
		}
		colName, _ := excelize.ColumnNumberToName(idx + 1)
		f.SetColWidth(sheetName, colName, colName, float64(maxLen+2))
	}
}

// SaveToRow 存入一行, rowNum 为行号（从1开始）
func SaveToRow(f *excelize.File, sheetName string, row []any, rowNum int) {
	for colIndex, v := range row {
		// 1,1单元格坐标  单元格: A1 (第A列, 第1行)
		cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowNum)
		f.SetCellValue(sheetName, cell, v)
	}
}
