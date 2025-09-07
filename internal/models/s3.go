package models

import (
	"time"

	"gorm.io/gorm"
)

type S3Data struct {
	gorm.Model

	ResourceID string  `gorm:"column:resource_id;type:varchar(64);not null;uniqueIndex:uidx_resource_id" json:"resource_id"` // 商业资源ID// 商业资源ID
	Bucket     string  `gorm:"column:bucket;type:varchar(256);not null" json:"bucket"`
	ObjectKey  string  `gorm:"column:object_key;type:varchar(512);not null" json:"object_key"`
	FileName   string  `gorm:"column:file_name;type:varchar(256);not null" json:"file_name"`
	FileSize   int64   `gorm:"column:file_size;not null" json:"file_size"`
	MimeType   string  `gorm:"column:mime_type;type:varchar(128);not null" json:"mime_type"`
	Tag        *string `gorm:"column:tag;type:json;" json:"tag"`
}
type S3Resource struct {
	ID uint `gorm:"primarykey"`

	ResourceID string `gorm:"type:varchar(64);not null;comment:资源ID;index:idx_resource_id" json:"resource_id"` // 资源ID // 资源ID
	URL        string `gorm:"type:varchar(512);not null;comment:资源URL" json:"url"`                             // 资源URL
	UserID     string `gorm:"type:varchar(64);not null;comment:用户ID;index:idx_user_id" json:"-"`               // 该资源URL所属的用户ID（微信OpenID），用于做请求定位
	Private    bool   `gorm:"type:boolean;not null;default:false;comment:是否私有" json:"private"`                 // 是否私有，私有资源只能用户自己访问

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	ExpiredAt gorm.DeletedAt `gorm:"index;comment:过期时间;column:expired_at"` // 过期时间
}

func (S3Resource) TableName() string {
	return "s3_resources"
}
func (S3Data) TableName() string {
	return "s3_data"
}
