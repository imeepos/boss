package apprelease

import "testing"

func mk(status string, code int, percent int, force bool, wl []int64, minSup int) Release {
	return Release{ID: 1, App: AppUser, Platform: PlatformAndroid, Version: "1.1.0",
		VersionCode: code, Status: status, RolloutPercent: percent, Force: force,
		WhitelistIDs: wl, MinSupportedCode: minSup, Notes: "n", Sha256: "abc", ApkSize: 12}
}

func TestDecideNoCandidate(t *testing.T) {
	got := Decide(5, "d", 0, nil)
	if got.UpdateAvailable {
		t.Fatalf("no candidate should be no update")
	}
}

func TestDecideUpToDate(t *testing.T) {
	got := Decide(11, "d", 0, []Release{mk(StatusPublished, 11, 0, false, nil, 1)})
	if got.UpdateAvailable {
		t.Fatalf("same versionCode should be no update")
	}
}

func TestDecidePublishedOptional(t *testing.T) {
	got := Decide(10, "d", 0, []Release{mk(StatusPublished, 11, 0, false, nil, 1)})
	if !got.UpdateAvailable || got.Force {
		t.Fatalf("want optional update, got %+v", got)
	}
	if got.Version != "1.1.0" || got.Sha256 != "abc" || got.Size != 12 || got.ReleaseID != 1 {
		t.Fatalf("projection mismatch: %+v", got)
	}
}

func TestDecideForceBelowMinSupported(t *testing.T) {
	got := Decide(2, "d", 0, []Release{mk(StatusPublished, 11, 0, false, nil, 10)})
	if !got.UpdateAvailable || !got.Force {
		t.Fatalf("below minSupportedCode must force, got %+v", got)
	}
}

func TestDecideForceFlag(t *testing.T) {
	got := Decide(10, "d", 0, []Release{mk(StatusPublished, 11, 0, true, nil, 1)})
	if !got.Force {
		t.Fatalf("force flag must propagate")
	}
}

func TestGrayBucketStable(t *testing.T) {
	// percent=100 全命中;同 deviceID 两次判定一致。
	r := mk(StatusGray, 11, 100, false, nil, 1)
	a := Decide(10, "device-a", 0, []Release{r})
	b := Decide(10, "device-a", 0, []Release{r})
	if !a.UpdateAvailable || !b.UpdateAvailable || a.ReleaseID != b.ReleaseID {
		t.Fatalf("full rollout must hit and be stable")
	}
}

func TestGrayZeroPercentWhitelistOnly(t *testing.T) {
	r := mk(StatusGray, 11, 0, false, []int64{7}, 1)
	if Decide(10, "any", 0, []Release{r}).UpdateAvailable {
		t.Fatalf("0%% gray without whitelist hit should not update")
	}
	if got := Decide(10, "any", 7, []Release{r}); !got.UpdateAvailable {
		t.Fatalf("whitelist subject must hit gray")
	}
}

func TestGrayMissFallsBackPublished(t *testing.T) {
	gray := mk(StatusGray, 12, 0, false, nil, 1)
	gray.ID = 2
	pub := mk(StatusPublished, 11, 0, false, nil, 1)
	pub.ID = 1
	// 无 PUBLISHED 时未命中灰度 → 无更新。
	if got := Decide(10, "x", 0, []Release{gray}); got.UpdateAvailable {
		t.Fatalf("gray miss without published fallback should be no update, got %+v", got)
	}
	// 有 PUBLISHED 时回落 PUBLISHED(即使 versionCode 更低)。
	got := Decide(10, "x", 0, []Release{gray, pub})
	if !got.UpdateAvailable || got.VersionCode != 11 || got.ReleaseID != 1 {
		t.Fatalf("gray miss must fall back to published, got %+v", got)
	}
}

func TestValidate(t *testing.T) {
	bad := Release{App: "x", Version: "1", VersionCode: 1, Status: StatusDraft, MinSupportedCode: 1}
	if err := bad.validate(); err == nil {
		t.Fatal("bad app must fail")
	}
	r := mk(StatusDraft, 5, 0, false, nil, 1)
	r.Platform = "" // 默认 android
	if err := r.validate(); err != nil || r.Platform != PlatformAndroid {
		t.Fatalf("platform default failed: %v", err)
	}
	r.RolloutPercent = 101
	if err := r.validate(); err == nil {
		t.Fatal("rollout>100 must fail")
	}
}
