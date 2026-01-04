package services

import (
	"context"
	"errors"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"gorm.io/gorm"
)

type PointsService struct {
	db *gorm.DB
}

func NewPointsService(db *gorm.DB) *PointsService {
	return &PointsService{
		db: db,
	}
}

// GetUserPoints 获取用户积分信息
func (s *PointsService) GetUserPoints(ctx context.Context, userID uint) (*response.UserPointsResponse, error) {
	var user models.User
	err := s.db.WithContext(ctx).Select("id, nickname, points").First(&user, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	return &response.UserPointsResponse{
		UserID: user.ID,
		User: &response.UserSimpleResponse{
			ID:       user.ID,
			Nickname: user.Nickname,
		},
		Points: user.Points,
	}, nil
}

// GetPointsTransactions 获取积分交易记录
func (s *PointsService) GetPointsTransactions(ctx context.Context, userID uint, req *request.GetPointsTransactionsRequest) (*response.PageResponse, error) {
	var transactions []models.PointsTransaction
	var total int64

	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.PointsTransaction{})

	// 普通用户只能看自己的记录
	if utils.IsAdmin(ctx) {
		if req.UserID != nil {
			query = query.Where("user_id = ?", *req.UserID)
		}
	} else {
		query = query.Where("user_id = ?", userID)
	}
	// 类型过滤
	if req.Type != nil {
		query = query.Where("type = ?", *req.Type)
	}

	// 来源过滤
	if req.Source != nil {
		query = query.Where("source = ?", *req.Source)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&transactions).Error; err != nil {
		return nil, err
	}

	// 批量获取用户信息（管理员查看时）
	isAdmin := utils.IsAdmin(ctx)
	userMap := make(map[uint]*response.UserSimpleResponse)
	if isAdmin {

		if len(transactions) > 0 {
			var userIDs []uint
			for _, transaction := range transactions {
				userIDs = append(userIDs, transaction.UserID)
			}

			var users []models.User
			if err := s.db.Select("id, nickname").Where("id IN ?", userIDs).Find(&users).Error; err == nil {
				for _, user := range users {
					userMap[user.ID] = &response.UserSimpleResponse{
						ID:       user.ID,
						Nickname: user.Nickname,
					}
				}
			}
		}
	}

	// 转换为响应格式
	var transactionResponses []response.PointsTransactionResponse
	for _, transaction := range transactions {
		// 使用预查询的用户信息
		var user *response.UserSimpleResponse
		if isAdmin && len(userMap) > 0 {
			user = userMap[transaction.UserID]
		}

		transactionResponses = append(transactionResponses, response.PointsTransactionResponse{
			ID:          transaction.ID,
			UserID:      transaction.UserID,
			User:        user,
			Type:        transaction.Type,
			Source:      transaction.Source,
			Points:      transaction.Points,
			Description: transaction.Description,
			RelatedID:   transaction.RelatedID,
			CreatedAt:   transaction.CreatedAt,
		})
	}

	return &response.PageResponse{
		Data:  transactionResponses,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// SpendPoints 消费积分（原RedeemPoints）
func (s *PointsService) SpendPoints(ctx context.Context, userID uint, req *request.SpendPointsRequest) error {
	// 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 检查积分是否足够
	if user.Points < req.Points {
		return errors.New("积分不足")
	}

	// 开启事务
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 扣除积分
		if err := tx.Model(&user).Update("points", gorm.Expr("points - ?", req.Points)).Error; err != nil {
			return err
		}

		// 记录交易
		transaction := models.PointsTransaction{
			UserID:      userID,
			Type:        models.PointsTransactionTypeSpend,
			Source:      models.PointsTransactionSourceRedeem,
			Points:      -int(req.Points), // 负数表示扣除
			Description: req.Description,
		}

		return tx.Create(&transaction).Error
	})
}

// AddPoints 增加积分（内部方法，用于其他服务调用）
func (s *PointsService) AddPoints(ctx context.Context, tx *gorm.DB, userID uint, points int, source string, description string, relatedID *uint) error {
	// 获取用户信息
	var user models.User
	if err := tx.WithContext(ctx).First(&user, userID).Error; err != nil {
		return err
	}

	// 更新积分
	if err := tx.WithContext(ctx).Model(&user).Update("points", gorm.Expr("points + ?", points)).Error; err != nil {
		return err
	}

	// 记录交易
	transaction := models.PointsTransaction{
		UserID:      userID,
		Type:        models.PointsTransactionTypeEarn,
		Source:      source,
		Points:      points,
		Description: description,
		RelatedID:   relatedID,
	}

	return tx.Create(&transaction).Error
}

// GetUserPointsStats 获取用户积分统计
func (s *PointsService) GetUserPointsStats(ctx context.Context, userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取用户积分
	userPoints, err := s.GetUserPoints(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats["points"] = userPoints.Points

	// 获取排名
	var rank int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).
		Where("points > (SELECT points FROM users WHERE id = ?)", userID).
		Count(&rank).Error; err != nil {
		return nil, err
	}
	stats["rank"] = rank + 1

	// 投稿获得积分总数
	var contributionPoints int64
	if err := s.db.WithContext(ctx).Model(&models.PointsTransaction{}).
		Where("user_id = ? AND type = ? AND source = ?",
			userID, models.PointsTransactionTypeEarn, models.PointsTransactionSourceContribution).
		Select("COALESCE(SUM(points), 0)").
		Scan(&contributionPoints).Error; err != nil {
		return nil, err
	}
	stats["contribution_points"] = contributionPoints

	// 兑换使用积分总数
	var redeemPoints int64
	if err := s.db.WithContext(ctx).Model(&models.PointsTransaction{}).
		Where("user_id = ? AND type = ? AND source = ?",
			userID, models.PointsTransactionTypeSpend, models.PointsTransactionSourceRedeem).
		Select("COALESCE(SUM(ABS(points)), 0)").
		Scan(&redeemPoints).Error; err != nil {
		return nil, err
	}
	stats["redeem_points"] = redeemPoints

	return stats, nil
}

// GrantPoints 管理员手动赋予积分
func (s *PointsService) GrantPoints(ctx context.Context, userID uint, points int, description string) error {
	// 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("用户不存在")
		}
		return err
	}

	// 开启事务
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新积分（支持正数和负数）
		newPoints := int(user.Points) + points
		if newPoints < 0 {
			return errors.New("积分不足，无法扣除")
		}

		if err := tx.Model(&user).Update("points", newPoints).Error; err != nil {
			return err
		}

		// 记录交易
		transactionType := models.PointsTransactionTypeEarn
		if points < 0 {
			transactionType = models.PointsTransactionTypeSpend
		}

		transaction := models.PointsTransaction{
			UserID:      userID,
			Type:        transactionType,
			Source:      models.PointsTransactionSourceAdminGrant,
			Points:      points,
			Description: description,
		}

		return tx.Create(&transaction).Error
	})
}
