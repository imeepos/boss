package odn

import "testing"

// 勘测任务纯校验(W7):标题边界/建议枚举/坐标范围/幂等键长度/上报人代次。

func TestValidateSurveyCreate(t *testing.T) {
	if err := ValidateSurveyCreate(SurveyCreateInput{Title: "网格A勘测"}); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := ValidateSurveyCreate(SurveyCreateInput{}); err == nil {
		t.Fatal("empty title must fail")
	}
	if err := ValidateSurveyCreate(SurveyCreateInput{Title: string(make([]byte, 129))}); err == nil {
		t.Fatal("long title must fail")
	}
}

func TestValidateSurveyReport(t *testing.T) {
	base := SurveyReport{TaskID: 1, WorkerID: 7, Suggestion: SuggestCanInstall, ClientMsgID: "msg-12345678"}
	if err := ValidateSurveyReport(base); err != nil {
		t.Fatalf("valid: %v", err)
	}
	bad := base
	bad.Suggestion = "MAYBE"
	if err := ValidateSurveyReport(bad); err == nil {
		t.Fatal("bad suggestion must fail")
	}
	coord := base
	coord.Lat = 95
	if err := ValidateSurveyReport(coord); err == nil {
		t.Fatal("lat out of range must fail")
	}
	idp := base
	idp.ClientMsgID = "short"
	if err := ValidateSurveyReport(idp); err == nil {
		t.Fatal("short clientMsgId must fail")
	}
	noWorker := base
	noWorker.WorkerID = 0
	if err := ValidateSurveyReport(noWorker); err == nil {
		t.Fatal("missing worker must fail")
	}
}

func TestValidateProgressReporterType(t *testing.T) {
	w := ProgressEntry{FacilityCode: "P01001", ClientMsgID: "msg-12345678",
		ReporterType: ProgressReporterWorker}
	if err := ValidateProgressEntry(w); err == nil {
		t.Fatal("worker reporter without id must fail")
	}
	w.ReportedBy = 6
	if err := ValidateProgressEntry(w); err != nil {
		t.Fatalf("worker reporter valid: %v", err)
	}
	bad := ProgressEntry{FacilityCode: "P01001", ClientMsgID: "msg-12345678", ReporterType: "ADMIN"}
	if err := ValidateProgressEntry(bad); err == nil {
		t.Fatal("unknown reporter type must fail")
	}
	a := ProgressEntry{FacilityCode: "P01001", ClientMsgID: "msg-12345678",
		ReporterType: ProgressReporterAccount, ReportedBy: 3}
	if err := ValidateProgressEntry(a); err != nil {
		t.Fatalf("account reporter valid: %v", err)
	}
}
