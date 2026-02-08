// Package perf provides a Just-In-Time compilation mechanism for the sonic JSON library,
//
//	allowing for faster JSON serialization and deserialization by pre-compiling the necessary types at runtime.
//	This can significantly improve performance when working with JSON data in the application.
//
// Copyright 2024 The Together For Study Authors. All rights reserved.
package perf

import (
	"reflect"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/bytedance/sonic"
)

func init() {
	logger.Infoln("Init Just-In-Time Compilation for Sonic")
	_ = sonic.Pretouch(reflect.TypeOf(&request.WechatLoginRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.GetQuestionRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.RecordStudyRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.SubmitPracticeRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.GetProjectUsageRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.GetNotificationsRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.MaterialListRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.MaterialListRequest{}))
	_ = sonic.Pretouch(reflect.TypeOf(&request.MaterialSearchRequest{}))

	_ = sonic.Pretouch(reflect.TypeOf(&response.CountdownResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.CourseTableResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.ClassInfo{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.SearchClassResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.FailRateListResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.MaterialListResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.MaterialDetailResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.MaterialDescResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.MaterialCategoryResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.MaterialLogResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.TopMaterialsResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.MaterialStatsResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.NotificationResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.NotificationSimpleResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.PageResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.UserPointsResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.PomodoroRankingItem{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.QuestionProjectResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.QuestionListResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.QuestionResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.SystemOnlineStatResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.ProjectOnlineStatResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.StudyTaskResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.WechatLoginResponse{}))
	_ = sonic.Pretouch(reflect.TypeOf(&response.WechatSession{}))

	logger.Info("Just-In-Time Compilation for Sonic complete")
}
