package workerapi

import (
	"math"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

type locationReportReq struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	AccuracyM   float32 `json:"accuracyM"`
	SpeedMPS    float32 `json:"speedMps"`
	Bearing     float32 `json:"bearing"`
	TimestampMs int64   `json:"timestampMs"`
}

func registerWorkerLocationRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/location/report", workerLocationReportHandler(a))
}

func workerLocationReportHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		var req locationReportReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := validateLocation(req); err != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": err.Error()})
			return
		}
		reportedAt := clock.Now()
		if req.TimestampMs > 0 {
			reportedAt = time.UnixMilli(req.TimestampMs)
		}
		if err := a.WorkerLocation.ReportLocation(c.Request.Context(), worker.Location{
			WorkerID: workerID, Lat: req.Lat, Lng: req.Lng, AccuracyM: req.AccuracyM,
			SpeedMPS: req.SpeedMPS, Bearing: req.Bearing, ReportedAt: reportedAt,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "reportedAt": reportedAt})
	}
}

func validateLocation(req locationReportReq) error {
	if math.IsNaN(req.Lat) || math.IsInf(req.Lat, 0) || req.Lat < -90 || req.Lat > 90 {
		return errInvalidLocation("lat")
	}
	if math.IsNaN(req.Lng) || math.IsInf(req.Lng, 0) || req.Lng < -180 || req.Lng > 180 {
		return errInvalidLocation("lng")
	}
	if req.AccuracyM < 0 || req.SpeedMPS < 0 || req.Bearing < 0 || req.Bearing >= 360 {
		return errInvalidLocation("measurement")
	}
	return nil
}

func errInvalidLocation(field string) error { return &invalidLocationError{field: field} }

type invalidLocationError struct{ field string }

func (e *invalidLocationError) Error() string { return "invalid location " + e.field }
