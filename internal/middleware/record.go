package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequestRecordMiddleware 通用请求记录中间件
// 功能：
// 1. 每日登录积分奖励（保留原有功能）
// 2. 系统在线人数统计（TTL 1分钟）
func RequestRecordMiddleware(db *gorm.DB, pointsService *services.PointsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只处理已认证的用户
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			c.Next()
			return
		}

		// 如果Redis不可用，跳过统计功能
		if cache.GlobalCache == nil {
			// 即使Redis不可用，也继续执行每日登录逻辑（如果有数据库）
			c.Next()
			return
		}

		ctx := c.Request.Context()
		userIDStr := strconv.FormatUint(uint64(uid), 10)

		// ==================== 每日登录积分奖励（保留原有功能） ====================
		// 获取今天的日期（以服务器时间为准）
		today := time.Now().Format("2006-01-02")
		redisKey := fmt.Sprintf("daily_login:%s", today)

		// 原子性检查并添加：使用SAdd的返回值判断
		// 如果返回1，说明是新添加的（之前不存在）
		// 如果返回0，说明已经存在
		added, err := cache.GlobalCache.SAdd(ctx, redisKey, userIDStr)
		if err == nil && added > 0 {
			// 如果是新添加的（added > 0），需要记录到数据库并加积分
			// 在事务中记录UserActivity并加积分
			err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				// 解析日期
				date, err := time.Parse("2006-01-02", today)
				if err != nil {
					return err
				}

				// 使用ON DUPLICATE KEY UPDATE处理唯一约束
				// GORM的Clauses可以处理，但更简单的方式是先查询
				var activity models.UserActivity
				err = tx.Where("user_id = ? AND date = ?", uid, date).First(&activity).Error

				if err == gorm.ErrRecordNotFound {
					// 创建新记录
					activity = models.UserActivity{
						UserID:     uid,
						Date:       date,
						VisitCount: 1,
					}
					if err := tx.Create(&activity).Error; err != nil {
						return err
					}
				} else if err != nil {
					return err
				} else {
					// 更新访问次数
					if err := tx.Model(&activity).Update("visit_count", gorm.Expr("visit_count + 1")).Error; err != nil {
						return err
					}
				}

				// 加登录积分（每日登录 +10积分）
				return pointsService.AddPoints(ctx, tx, uid, 10,
					models.PointsTransactionSourceDailyLogin, "每日登录", nil)
			})

			if err != nil {
				// 记录错误但不影响主流程
				// 错误已通过事务回滚处理
			}
		}

		// ==================== 系统在线人数统计（每个用户独立TTL 1分钟） ====================
		// 使用 Sorted Set，score 存储时间戳
		// 每次用户请求时刷新其时间戳，查询时只统计最近1分钟内的用户
		systemOnlineKey := "online:system"
		now := float64(time.Now().Unix())
		_ = cache.GlobalCache.ZAdd(ctx, systemOnlineKey, now, userIDStr)

		c.Next()
	}
}
