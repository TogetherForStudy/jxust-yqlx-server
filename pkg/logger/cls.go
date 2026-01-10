package logger

import (
	"context"
	"sync"
	"time"

	tencentcloudclssdkgo "github.com/tencentcloud/tencentcloud-cls-sdk-go"
)

var (
	logChannel       = make(chan *tencentcloudclssdkgo.Log, 1000)
	callBack         = &Callback{}
	producerInstance *tencentcloudclssdkgo.AsyncProducerClient
	producerMu       sync.Mutex
	shutdownOnce     sync.Once
	clsEnabled       bool
)

// TencentClsLoggerInit 腾讯云日志服务初始化
//
//	ctx: 控制日志发送协程的生命周期
//	endpoint: 日志服务的接入点地址，例如：ap-guangzhou.cls.tencentcs.com，具体请参见：https://cloud.tencent.com/document/product/614/18940#.E5.9F.9F.E5.90.8D
//	topicId: 日志主题ID，例如：topic-3n6jnk4z
//	secretId: 腾讯云API密钥ID
//	secretKey: 腾讯云API密钥Key
func TencentClsLoggerInit(ctx context.Context, enable bool,
	endpoint, topicId, secretId, secretKey string) error {
	clsEnabled = enable
	if !clsEnabled {
		return nil
	}

	producerConfig := tencentcloudclssdkgo.GetDefaultAsyncProducerClientConfig()
	// 填入域名信息，填写指引：https://cloud.tencent.com/document/product/614/18940#.E5.9F.9F.E5.90.8D，请参见链接中 API 上传日志 Tab 中的域名
	producerConfig.Endpoint = endpoint

	// 填入云API密钥信息。密钥信息获取请前往：https://console.cloud.tencent.com/cam/capi
	// 并请确保密钥关联的账号具有相应的日志上传权限，权限配置指引：https://cloud.tencent.com/document/product/614/68374#.E4.BD.BF.E7.94.A8-api-.E4.B8.8A.E4.BC.A0.E6.95.B0.E6.8D.AE
	producerConfig.AccessKeyID = secretId
	producerConfig.AccessKeySecret = secretKey

	// 创建异步生产者客户端实例
	instance, err := tencentcloudclssdkgo.NewAsyncProducerClient(producerConfig)
	if err != nil {
		return err
	}

	producerMu.Lock()
	producerInstance = instance
	producerMu.Unlock()

	go func() {
		// 启动异步发送程序
		producerInstance.Start()
		defer producerInstance.Close(60000)
		for {
			select {
			case log := <-logChannel:
				err = producerInstance.SendLog(topicId, log, callBack)
				if err != nil {
					Errorln(err.Error())
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// ShutdownLogger gracefully shuts down the logger, ensuring all logs are flushed
// timeout: maximum time to wait for logs to be flushed
func ShutdownLogger(timeout time.Duration) {
	shutdownOnce.Do(func() {
		// Create a deadline context
		deadline := time.Now().Add(timeout)

		// Close the log channel to prevent new logs
		close(logChannel)

		// Wait for the channel to be drained or timeout
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			if time.Now().After(deadline) {
				Warnf("Logger shutdown timeout reached, %d logs may be lost", len(logChannel))
				break
			}

			if len(logChannel) == 0 {
				break
			}

			<-ticker.C
		}

		// Close the producer instance
		producerMu.Lock()
		if producerInstance != nil {
			producerInstance.Close(timeout.Milliseconds())
		}
		producerMu.Unlock()

		// Sync zap logger
		if zlog != nil {
			_ = zlog.Sync()
		}
	})
}

type Callback struct {
}

func (callback *Callback) Success(result *tencentcloudclssdkgo.Result) {
	//attemptList := result.GetReservedAttempts()
	//for _, attempt := range attemptList {
	//	fmt.Printf("%+v \n", attempt)
	//}
}

func (callback *Callback) Fail(result *tencentcloudclssdkgo.Result) {
	Errorln(result.IsSuccessful(),
		result.GetErrorCode(),
		result.GetErrorMessage(),
		result.GetReservedAttempts(),
		result.GetRequestId(),
		result.GetTimeStampMs(),
	)
}
