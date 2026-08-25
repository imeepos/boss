package app

// gRPC provision/v1 服务端:下发任务入队/失败重试/状态查询(order→provisioner)。
// 契约字段 task_no/order_id/stage_event/template_code 已落地 provision_tasks(迁移 000032)。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	provisionv1 "github.com/ymm-001/boss/api/proto/boss/provision/v1"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// provisionGRPC ProvisionService 实现。
type provisionGRPC struct {
	provisionv1.UnimplementedProvisionServiceServer
	prov   provision.ProvisionService
	ord    order.OrderService
	aaaSvc aaa.AaaService
}

// provErrMap 下发域错误 → 契约错误码。
var provErrMap = map[error]apitypes.Code{
	provision.ErrTaskNotFound:      apitypes.CodeNotFound,
	provision.ErrIllegalTransition: apitypes.CodeStateInvalid,
	order.ErrOrderNotFound:         apitypes.CodeNotFound,
	aaa.ErrNotFound:                apitypes.CodeNotFound,
}

// EnqueueTask 下发任务入队:模板编码 → 模板,订单 → LO 账号,任务落库 PENDING。
func (s *provisionGRPC) EnqueueTask(ctx context.Context, req *provisionv1.Task) (*provisionv1.OpResponse, error) {
	templateID, err := s.templateIDByCode(ctx, req.TemplateCode)
	if err != nil {
		return &provisionv1.OpResponse{Code: grpcCodeFor(err, provErrMap)}, nil
	}
	if templateID == 0 || req.OrderId <= 0 {
		return &provisionv1.OpResponse{Code: commonv1.Code_CODE_INVALID_PARAM}, nil
	}
	loAccountID, err := s.loAccountIDByOrder(ctx, req.OrderId)
	if err != nil {
		return &provisionv1.OpResponse{Code: grpcCodeFor(err, provErrMap)}, nil
	}
	taskNo := req.TaskNo
	if taskNo == "" {
		taskNo = fmt.Sprintf("TASK-%d", time.Now().UnixNano())
	}
	if _, err := s.prov.CreateTask(ctx, provision.Task{
		TaskNo: taskNo, OrderID: req.OrderId, StageEvent: req.StageEvent,
		LoAccountID: loAccountID, TemplateID: templateID, Status: "PENDING",
	}); err != nil {
		return &provisionv1.OpResponse{Code: commonv1.Code_CODE_INTERNAL}, nil
	}
	return &provisionv1.OpResponse{Code: commonv1.Code_CODE_OK, TaskNo: taskNo, Status: "PENDING"}, nil
}

func validProvisionStage(stage string) bool {
	switch stage {
	case "preConfigOLT", "activateUser", "notifyActivation":
		return true
	default:
		return false
	}
}

// RetryTask 失败重试:FAILED→PENDING + 重试计数留痕。
func (s *provisionGRPC) RetryTask(ctx context.Context, req *provisionv1.RetryTaskRequest) (*provisionv1.OpResponse, error) {
	tk, err := s.prov.GetTaskByNo(ctx, req.TaskNo)
	if err != nil {
		return &provisionv1.OpResponse{Code: grpcCodeFor(err, provErrMap)}, nil
	}
	logs, _ := s.prov.ListLogs(ctx, tk.ID)
	var max int16
	for _, l := range logs {
		if l.Retries > max {
			max = l.Retries
		}
	}
	if err := s.prov.RetryTask(ctx, tk.ID, max); err != nil {
		return &provisionv1.OpResponse{Code: grpcCodeFor(err, provErrMap)}, nil
	}
	return &provisionv1.OpResponse{Code: commonv1.Code_CODE_OK, TaskNo: req.TaskNo, Status: "PENDING"}, nil
}

// GetTask 任务状态查询(轮询/回调校验);无 code 字段,失败走 gRPC status。
func (s *provisionGRPC) GetTask(ctx context.Context, req *provisionv1.GetTaskRequest) (*provisionv1.Task, error) {
	tk, err := s.prov.GetTaskByNo(ctx, req.TaskNo)
	if err != nil {
		if errors.Is(err, provision.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "provision: task not found")
		}
		return nil, status.Error(codes.Internal, "provision: get task failed")
	}
	return &provisionv1.Task{
		OrderId: tk.OrderID, TaskNo: tk.TaskNo, StageEvent: tk.StageEvent,
		TemplateCode: s.templateCodeByID(ctx, tk.TemplateID), Params: map[string]string{},
	}, nil
}

// templateCodeByID 模板 ID → 模板编码(GetTask 透出);未知返回空。
func (s *provisionGRPC) templateCodeByID(ctx context.Context, id int64) string {
	list, err := s.prov.ListTemplates(ctx)
	if err != nil {
		return ""
	}
	for _, t := range list {
		if t.ID == id {
			return t.Code
		}
	}
	return ""
}

// templateIDByCode 模板编码 → 模板 ID;未知返回 0。
func (s *provisionGRPC) templateIDByCode(ctx context.Context, code string) (int64, error) {
	list, err := s.prov.ListTemplates(ctx)
	if err != nil {
		return 0, err
	}
	for _, t := range list {
		if t.Code == code {
			return t.ID, nil
		}
	}
	return 0, nil
}

// loAccountIDByOrder 订单 → LO 账号(orders.customer_id → lo_accounts)。
func (s *provisionGRPC) loAccountIDByOrder(ctx context.Context, orderID int64) (int64, error) {
	o, _, err := s.ord.Track(ctx, orderID)
	if err != nil {
		return 0, err
	}
	lo, err := s.aaaSvc.GetLoAccountByCustomer(ctx, o.CustomerID)
	if err != nil {
		return 0, err
	}
	return lo.ID, nil
}
