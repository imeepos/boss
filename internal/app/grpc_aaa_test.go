package app

// gRPC aaa/v1 契约单测:停复机即时生效、授权查询(停机亦返回状态)、话单入账。

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	aaav1 "github.com/ymm-001/boss/api/proto/boss/aaa/v1"
	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
)

type stubAaaSvc struct {
	aaa.AaaService
	suspendErr, resumeErr error
	lo                    *aaa.LoAccount
	loErr                 error
}

func (s *stubAaaSvc) SuspendLoAccount(context.Context, int64) error { return s.suspendErr }
func (s *stubAaaSvc) ResumeLoAccount(context.Context, int64) error  { return s.resumeErr }
func (s *stubAaaSvc) GetLoAccountByLoid(_ context.Context, _ string) (*aaa.LoAccount, error) {
	return s.lo, s.loErr
}

type stubAuthorizer struct {
	dec aaa.Decision
	err error
}

func (s *stubAuthorizer) Decide(_ context.Context, _ string) (aaa.Decision, error) {
	return s.dec, s.err
}

type stubEmitter struct {
	err error
	got aaability.CDR
}

func (s *stubEmitter) Emit(_ context.Context, cdr aaability.CDR) error {
	s.got = cdr
	return s.err
}

func aaaGRPCWith(svc *stubAaaSvc, auth *stubAuthorizer, cdr *stubEmitter) *aaaGRPC {
	return &aaaGRPC{aaaSvc: svc, auth: auth, cdr: cdr}
}

func TestAaaGRPC_SuspendResume(t *testing.T) {
	ctx := context.Background()

	t.Run("停机即时生效", func(t *testing.T) {
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(&stubAaaSvc{}, &stubAuthorizer{}, &stubEmitter{}))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).Suspend(ctx, &aaav1.SuspendRequest{LoAccountId: 1, Reason: "ARREARS"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || !resp.Effective {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("非法停机 40910", func(t *testing.T) {
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(&stubAaaSvc{suspendErr: aaa.ErrIllegalTransition}, &stubAuthorizer{}, &stubEmitter{}))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).Suspend(ctx, &aaav1.SuspendRequest{LoAccountId: 1})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_STATE_INVALID {
			t.Fatalf("code=%v, want STATE_INVALID", resp.Code)
		}
	})

	t.Run("复机即时生效", func(t *testing.T) {
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(&stubAaaSvc{}, &stubAuthorizer{}, &stubEmitter{}))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).Resume(ctx, &aaav1.ResumeRequest{LoAccountId: 1})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || !resp.Effective {
			t.Fatalf("resp=%+v", resp)
		}
	})
}

func TestAaaGRPC_GetAuthorization(t *testing.T) {
	ctx := context.Background()

	t.Run("在服账号返回带宽与QoS", func(t *testing.T) {
		svc := &stubAaaSvc{lo: &aaa.LoAccount{Status: "ACTIVE"}}
		auth := &stubAuthorizer{dec: aaa.Decision{Authorize: true, Bandwidth: "300M/150M", QosTemplate: "QOS-HS"}}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(svc, auth, &stubEmitter{}))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).GetAuthorization(ctx, &aaav1.GetAuthorizationRequest{Loid: "LOID-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || resp.Status != "ACTIVE" || resp.Bandwidth != "300M/150M" || resp.QosTemplate != "QOS-HS" {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("停机账号仍返回状态", func(t *testing.T) {
		svc := &stubAaaSvc{lo: &aaa.LoAccount{Status: "SUSPENDED"}}
		auth := &stubAuthorizer{err: aaa.ErrSuspended}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(svc, auth, &stubEmitter{}))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).GetAuthorization(ctx, &aaav1.GetAuthorizationRequest{Loid: "LOID-1"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || resp.Status != "SUSPENDED" {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("账号不存在 404", func(t *testing.T) {
		svc := &stubAaaSvc{loErr: aaa.ErrNotFound}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(svc, &stubAuthorizer{}, &stubEmitter{}))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).GetAuthorization(ctx, &aaav1.GetAuthorizationRequest{Loid: "LOID-NOPE"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_NOT_FOUND {
			t.Fatalf("code=%v, want NOT_FOUND", resp.Code)
		}
	})
}

func TestAaaGRPC_EmitCDR(t *testing.T) {
	ctx := context.Background()

	t.Run("Stop 话单含时长", func(t *testing.T) {
		em := &stubEmitter{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(&stubAaaSvc{}, &stubAuthorizer{}, em))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).EmitCDR(ctx, &aaav1.CDR{
			Loid: "LOID-1", SessionId: "S-1", EventType: "Stop",
			InputOctets: 100, OutputOctets: 200, StartedAt: 1000, FinishedAt: 1060,
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK {
			t.Fatalf("resp=%+v", resp)
		}
		if em.got.AcctStatus != 2 || em.got.SessionTime != 60 || em.got.InputOctets != 100 {
			t.Fatalf("cdr=%+v", em.got)
		}
	})

	t.Run("投递失败 50200", func(t *testing.T) {
		em := &stubEmitter{err: context.DeadlineExceeded}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			aaav1.RegisterAaaServiceServer(s, aaaGRPCWith(&stubAaaSvc{}, &stubAuthorizer{}, em))
		})
		resp, err := aaav1.NewAaaServiceClient(conn).EmitCDR(ctx, &aaav1.CDR{Loid: "LOID-1", EventType: "Start"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_DOWNSTREAM_ERR {
			t.Fatalf("code=%v, want DOWNSTREAM_ERR", resp.Code)
		}
	})
}
