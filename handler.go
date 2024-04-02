package xlst

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	defaultImage = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII="

	pixelCoefficient = 1.34
)

func (t *templater) fieldNameKyeHandler(file *excelize.File, sheet string, rowIdx, colIdx *int, value interface{}) error {
	axis, _ := excelize.CoordinatesToCellName(*colIdx+1, *rowIdx+1)
	return file.SetCellValue(sheet, axis, value)
}

func (t *templater) tableKeyHandler(file *excelize.File, sheet string, rowIdx, colIdx *int, value interface{}) error {
	array, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("tableKeyHandler: wrong type payload, array type expected")
	}

	if err := file.RemoveRow(sheet, *rowIdx+1); err != nil {
		return fmt.Errorf("remove row number %d: %w", *rowIdx+1, err)
	}

	hRowNumb := *rowIdx + 1
	rows, _ := file.GetRows(sheet)
	hRow := rows[hRowNumb-1]

	for i, item := range array {
		if i < len(array)-1 {
			if err := file.DuplicateRowTo(sheet, hRowNumb, hRowNumb+1+i); err != nil {
				return fmt.Errorf("dublicate row number %d: %w", hRowNumb, err)
			}
		}
		for j := *colIdx; j < len(hRow); j++ {
			placeholderType, v, err := t.placeholder.GetValue(item, hRow[j])
			if err != nil {
				return err
			}
			if keyHandler, ok := t.keyHandler[placeholderType]; ok {
				rowIdx := hRowNumb + i - 1
				keyHandler(file, sheet, &rowIdx, &j, v)
			}
		}
	}

	// TODO:
	// deleting title of table
	// if len(array) == 0 && *rowIdx != 0 {
	// 	file.RemoveRow(sheet, *rowIdx)
	// 	*rowIdx--
	// }
	*rowIdx = *rowIdx + len(array) - 1
	*colIdx = 0
	return nil
}

func (t *templater) qrCodeHandler(file *excelize.File, sheet string, rowIdx, colIdx *int, value interface{}) (err error) {
	rowHeight, _ := file.GetRowHeight(sheet, *rowIdx+1)
	qrcodePixels := pixelCoefficient * rowHeight

	str, ok := value.(string)
	if !ok {
		err = fmt.Errorf("qrCodeHandler: wrong type elements of array, string  type expected")
		return
	}

	var data []byte
	if data, err = t.qrcodeEncode(str, int(qrcodePixels)); err != nil {
		err = fmt.Errorf("qrCodeHandler: qrcode generate %s", err)
		return
	}
	axis, _ := excelize.CoordinatesToCellName(*colIdx+1, *rowIdx+1)
	file.SetCellValue(sheet, axis, "")
	if err = file.AddPictureFromBytes(sheet, axis, &excelize.Picture{Extension: ".png", File: data}); err != nil {
		err = fmt.Errorf("qrCodeHandler: insert qrcode to file %s", err)
	}
	return
}

func (t *templater) qrCodeRowHandler(file *excelize.File, sheet string, rowIdx, colIdx *int, value interface{}) (err error) {
	qrcodeDataArr, ok := value.([]interface{})
	if !ok {
		err = fmt.Errorf("qrCodeListHandler: wrong type payload, array type expected")
		return
	}

	for _, qrcodeData := range qrcodeDataArr {
		t.qrCodeHandler(file, sheet, rowIdx, colIdx, qrcodeData)
		axis, _ := excelize.CoordinatesToCellName(*colIdx+1, *rowIdx+1)
		colNum, _, _ := getNumMergeCell(file, sheet, axis)
		*colIdx += colNum
	}
	return
}

func (s *templater) imageHandler(file *excelize.File, sheet string, rowIdx, colIdx *int, value interface{}) error {
	image, ok := value.(string)
	if !ok {
		return fmt.Errorf("imageHandler: wrong type payload, string type expected")
	}
	i := strings.Index(image, ",")
	image = image[i+1:]
	if len(image) == 0 {
		image = defaultImage
	}
	imageBytes, err := base64.StdEncoding.DecodeString(image)
	if err != nil {
		return fmt.Errorf("imageHandler: decode image %s", err)
	}

	axis, _ := excelize.CoordinatesToCellName(*colIdx+1, *rowIdx+1)
	file.SetCellValue(sheet, axis, "")
	if err := file.AddPictureFromBytes(sheet, axis, &excelize.Picture{Extension: ".png", File: imageBytes}); err != nil {
		return fmt.Errorf("imageHandler: insert image to file %s", err)
	}
	return nil
}

func getNumMergeCell(file *excelize.File, sheet string, axis string) (colNum int, rowNum int, err error) {
	mergedCells, err := file.GetMergeCells(sheet)
	if err != nil {
		return
	}
	for _, mergetCell := range mergedCells {
		if mergetCell.GetStartAxis() == axis {
			col1, row1, _ := excelize.CellNameToCoordinates(mergetCell.GetStartAxis())
			col2, row2, _ := excelize.CellNameToCoordinates(mergetCell.GetEndAxis())
			colNum, rowNum = col2-col1+1, row2-row1+1
			return
		}
	}
	colNum = 1
	rowNum = 1
	return
}
