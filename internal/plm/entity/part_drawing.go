package entity

import "time"

// PartDrawing 零件图纸版本管理
type PartDrawing struct {
	ID                string    `json:"id" gorm:"primaryKey;size:32"`
	BOMItemID         string    `json:"bom_item_id" gorm:"size:32;not null;index"`
	DrawingType       string    `json:"drawing_type" gorm:"size:4;not null"` // 2D / 3D
	Version           string    `json:"version" gorm:"size:16;not null"`     // v1, v2, v3...
	FileID            string    `json:"file_id" gorm:"size:32;not null"`
	FileName          string    `json:"file_name" gorm:"size:256;not null"`
	FileSize          int64     `json:"file_size" gorm:"default:0"`
	FileURL           string    `json:"file_url,omitempty" gorm:"size:512"`
	ChangeDescription string    `json:"change_description,omitempty" gorm:"type:text"`
	ChangeReason      string    `json:"change_reason,omitempty" gorm:"size:256"`
	UploadedBy        string    `json:"uploaded_by" gorm:"size:32;not null"`
	CreatedAt         time.Time `json:"created_at"`

	// Relations
	BOMItem  *ProjectBOMItem `json:"bom_item,omitempty" gorm:"foreignKey:BOMItemID"`
	Uploader *User           `json:"uploader,omitempty" gorm:"foreignKey:UploadedBy"`
}

func (PartDrawing) TableName() string {
	return "part_drawings"
}
