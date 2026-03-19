package services

import (
	"context"
	"errors"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DictionaryService struct {
	db *gorm.DB
}

func NewDictionaryService(db *gorm.DB) *DictionaryService {
	return &DictionaryService{db: db}
}

// GetRandomWord 从数据库随机获取一个词
func (s *DictionaryService) GetRandomWord(ctx context.Context) (*models.Dictionary, error) {
	var word models.Dictionary

	// 在 MySQL 中，可以使用 ORDER BY RAND() 取一条记录
	// 如果数据量巨大，可以考虑使用更高效的随机算法，但对于词典表，RAND() 通常足够
	err := s.db.WithContext(ctx).Model(&models.Dictionary{}).Order("RAND()").First(&word).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 词典表为空时返回占位词条，避免前端和 E2E 因空库直接失败。
			return &models.Dictionary{
				Word:       "暂无词条",
				Trans:      datatypes.JSON([]byte(`["词典数据建设中"]`)),
				Sentences:  datatypes.JSON([]byte(`[]`)),
				Phrases:    datatypes.JSON([]byte(`[]`)),
				Synos:      datatypes.JSON([]byte(`[]`)),
				RelWords:   datatypes.JSON([]byte(`[]`)),
				Source:     "system",
				PhoneticUK: "",
				PhoneticUS: "",
			}, nil
		}
		return nil, apperr.Wrap(constant.DictionaryRandomWordFailed, err)
	}

	return &word, nil
}
