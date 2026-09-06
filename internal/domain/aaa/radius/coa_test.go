package radius

// CoA/Disconnect 组包回归(RFC 5176):code 与最小定位属性集。

import (
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

func TestBuildDisconnectPacket(t *testing.T) {
	pkt := buildDisconnectPacket([]byte("secret"), "LOID-9", "S-9")
	if pkt.Code != radius.CodeDisconnectRequest {
		t.Fatalf("code=%v", pkt.Code)
	}
	if got := rfc2865.UserName_GetString(pkt); got != "LOID-9" {
		t.Fatalf("user-name=%q", got)
	}
	if got := rfc2866.AcctSessionID_GetString(pkt); got != "S-9" {
		t.Fatalf("session-id=%q", got)
	}
}
