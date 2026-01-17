package services

import (
	"context"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
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
		return nil, err
	}

	return &word, nil
}
