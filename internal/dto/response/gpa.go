package response

import (
	"time"

	"gorm.io/datatypes"
)

type GPABackupResponse struct {
	ID        uint           `json:"id"`
	Data      datatypes.JSON `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
