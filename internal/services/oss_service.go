package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
)

// OSSService 负责生成又拍云/自建 CDN 的 Token 防盗链参数
type OSSService struct {
	cfg *config.Config
}

func NewOSSService(cfg *config.Config) *OSSService {
	return &OSSService{cfg: cfg}
}

// GenerateToken 根据 URI 生成 _upt 参数与可选的完整签名 URL
// 算法参考: https://help.upyun.com/knowledge-base/cdn-token-limite/
func (s *OSSService) GenerateToken(uri string, ttlSeconds int64) (token string, expireAt int64, signedURL string, err error) {
	if uri == "" || !strings.HasPrefix(uri, "/") {
		return "", 0, "", fmt.Errorf("uri 必须以 / 开头，并且不可为空")
	}
	if s.cfg.UpyunTokenSecret == "" {
		return "", 0, "", fmt.Errorf("未配置 UPYUN_TOKEN_SECRET")
	}

	if ttlSeconds <= 0 {
		ttlSeconds = 600 // 默认 10 分钟
	}

	now := time.Now().Unix()
	expireAt = now + ttlSeconds

	// sign = MD5(secret & etime & URI)
	raw := fmt.Sprintf("%s&%d&%s", s.cfg.UpyunTokenSecret, expireAt, uri)
	sum := md5.Sum([]byte(raw))
	sign := hex.EncodeToString(sum[:])
	// _upt = sign{中间8位} + etime
	midStart := (len(sign) - 8) / 2
	mid := sign[midStart : midStart+8]
	token = fmt.Sprintf("%s%d", mid, expireAt)

	if base := strings.TrimSuffix(s.cfg.CdnBaseURL, "/"); base != "" {
		signedURL = fmt.Sprintf("%s%s?_upt=%s", base, uri, token)
	}
	return token, expireAt, signedURL, nil
}
