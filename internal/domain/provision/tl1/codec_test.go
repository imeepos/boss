package tl1

import (
	"errors"
	"reflect"
	"testing"
)

var (
	goldenA = "   HW_10.71.227.225 2010-07-27 14:32:06\nM  1 COMPLD\n   EN=0   ENDESC=成功。\n;"
	goldenB = "   HW_10.144.82.107 2010-11-26 09:54:45\nM  CTAG COMPLD\n   EN=0   ENDESC=成功。\n;"
	goldenC = "   HW_10.185.164.180 2016-06-23 09:54:12\nM  CTAG COMPLD\nTitle = list of ONU state\nONUID AdminState OperState AUTH AUTHINFO ONUIP LASTOFFTIME\n-----\n1     UP         UP        LOID  x\n;"
	goldenD = "   HW_10.185.164.180 2016-06-23 09:54:12\nM  CTAG COMPLD\nTitle = list of ONU state\nONUID AdminState OperState\n1     UP         UP\n>\nONUID AdminState OperState\n2     DOWN       DOWN\n;"
)

func TestParse_LoginOK(t *testing.T) {
	r, err := Parse(goldenA)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.CTag != "1" || r.Completion != "COMPLD" {
		t.Fatalf("ctag=%q compl=%q", r.CTag, r.Completion)
	}
	if r.EN != 0 {
		t.Fatalf("EN=%d want 0", r.EN)
	}
	if r.ENDESC != "成功。" {
		t.Fatalf("ENDESC=%q", r.ENDESC)
	}
	if len(r.Rows) != 0 {
		t.Fatalf("rows=%d want 0", len(r.Rows))
	}
	if r.Raw != goldenA {
		t.Fatalf("Raw not preserved verbatim")
	}
}

func TestParse_AddPONVLANOK(t *testing.T) {
	r, err := Parse(goldenB)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.CTag != "CTAG" || r.Completion != "COMPLD" || r.EN != 0 || r.ENDESC != "成功。" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestParse_Table(t *testing.T) {
	r, err := Parse(goldenC)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(r.Rows) != 1 {
		t.Fatalf("rows=%d want 1", len(r.Rows))
	}
	want := map[string]string{"ONUID": "1", "AdminState": "UP", "OperState": "UP", "AUTH": "LOID", "AUTHINFO": "x", "ONUIP": "", "LASTOFFTIME": ""}
	if !reflect.DeepEqual(r.Rows[0], want) {
		t.Fatalf("row=%#v want %#v", r.Rows[0], want)
	}
}

func TestParse_MultiBlock(t *testing.T) {
	r, err := Parse(goldenD)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(r.Rows) != 2 {
		t.Fatalf("rows=%d want 2", len(r.Rows))
	}
	if r.Rows[0]["ONUID"] != "1" || r.Rows[1]["ONUID"] != "2" {
		t.Fatalf("rows=%#v", r.Rows)
	}
	if r.Rows[1]["OperState"] != "DOWN" {
		t.Fatalf("row2 oper=%q", r.Rows[1]["OperState"])
	}
}

func TestParse_Empty(t *testing.T) {
	if _, err := Parse(""); !errors.Is(err, ErrParse) {
		t.Fatalf("want ErrParse got %v", err)
	}
}

func TestBuild_NoPayload(t *testing.T) {
	out, err := Build(Command{Verb: "LOGIN", Payload: []KV{{"UN", "admin"}, {"PWD", "x"}}}, "1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := "LOGIN:::1::UN=admin,PWD=x;"
	if string(out) != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestBuild_WithAccess(t *testing.T) {
	out, err := Build(Command{Verb: "ADD-PONVLAN", Access: []KV{{"OLTID", "10.144.82.107"}, {"PONID", "NA-0-7-5"}}, Payload: []KV{{"UN", "boss"}}}, "B1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := "ADD-PONVLAN::OLTID=10.144.82.107,PONID=NA-0-7-5:B1::UN=boss;"
	if string(out) != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestBuild_WhitelistReject(t *testing.T) {
	bad := []string{"a;b", "a,b", "a:b", "中文"}
	for _, v := range bad {
		_, err := Build(Command{Verb: "ADD-ONU", Payload: []KV{{"DESC", v}}}, "1")
		if !errors.Is(err, ErrBadParam) {
			t.Fatalf("value %q: want ErrBadParam got %v", v, err)
		}
	}
	if _, err := Build(Command{Verb: "AD", Payload: []KV{{"K,X", "v"}}}, "1"); !errors.Is(err, ErrBadParam) {
		t.Fatalf("want ErrBadParam got %v", err)
	}
}

// ---- 业务 ctag(Tag 位)黄金报文 ----

// TestBuild_TagGoldens 精确断言手册业务 ctag 报文:ADD-ONU=ADDONT、
// ADD-PONVLAN=Internet/TR069;非空 Tag 覆盖入参会话自增 ctag。
func TestBuild_TagGoldens(t *testing.T) {
	cases := []struct {
		name string
		c    Command
		ctag string
		want string
	}{
		{
			name: "ADD-ONU 手册 ctag ADDONT",
			c: Command{
				Verb: "ADD-ONU", Tag: "ADDONT",
				Access: []KV{{"OLTID", "10.0.0.9"}, {"PONID", "NA-0-7-5"}},
				Payload: []KV{
					{"AUTHTYPE", "LOID"}, {"ONUID", "loid-0001"},
					{"ONUNO", "7"}, {"DESC", "PRV-ORD-0001"}, {"ONUTYPE", "FTTH_E8C"},
				},
			},
			ctag: "B000001", // 会话自增值应被业务 Tag 覆盖
			want: "ADD-ONU::OLTID=10.0.0.9,PONID=NA-0-7-5:ADDONT::AUTHTYPE=LOID,ONUID=loid-0001,ONUNO=7,DESC=PRV-ORD-0001,ONUTYPE=FTTH_E8C;",
		},
		{
			name: "ADD-PONVLAN Internet 双层(带 SVLAN)",
			c: Command{
				Verb: "ADD-PONVLAN", Tag: "Internet",
				Access:  []KV{{"OLTID", "10.0.0.9"}, {"PONID", "NA-0-7-5"}, {"ONUIDTYPE", "LOID"}, {"ONUID", "loid-0001"}},
				Payload: []KV{{"SVLAN", "1000"}, {"CVLAN", "100"}, {"UV", "100"}, {"DESC", "PRV-ORD-0001-Internet"}, {"SCOS", "5"}},
			},
			ctag: "B000002",
			want: "ADD-PONVLAN::OLTID=10.0.0.9,PONID=NA-0-7-5,ONUIDTYPE=LOID,ONUID=loid-0001:Internet::SVLAN=1000,CVLAN=100,UV=100,DESC=PRV-ORD-0001-Internet,SCOS=5;",
		},
		{
			name: "ADD-PONVLAN TR069 单层(无 SVLAN)",
			c: Command{
				Verb: "ADD-PONVLAN", Tag: "TR069",
				Access:  []KV{{"OLTID", "10.0.0.9"}, {"PONID", "NA-0-7-5"}, {"ONUIDTYPE", "LOID"}, {"ONUID", "loid-0001"}},
				Payload: []KV{{"CVLAN", "100"}, {"UV", "100"}, {"DESC", "PRV-ORD-0001-TR069"}, {"SCOS", "5"}},
			},
			ctag: "B000003",
			want: "ADD-PONVLAN::OLTID=10.0.0.9,PONID=NA-0-7-5,ONUIDTYPE=LOID,ONUID=loid-0001:TR069::CVLAN=100,UV=100,DESC=PRV-ORD-0001-TR069,SCOS=5;",
		},
		{
			name: "空 Tag 仍用入参会话自增 ctag",
			c:    Command{Verb: "LST-ONU", Access: []KV{{"OLTID", "10.0.0.9"}}},
			ctag: "B000001",
			want: "LST-ONU::OLTID=10.0.0.9:B000001::;",
		},
	}
	for _, tc := range cases {
		out, err := Build(tc.c, tc.ctag)
		if err != nil {
			t.Fatalf("%s: Build: %v", tc.name, err)
		}
		if string(out) != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, out, tc.want)
		}
	}
}

// TestBuild_TagWhitelistReject ctag 白名单外的 Tag 一律 ErrBadParam,
// 拒 : ; , = 空格 等分隔/注入字符,报文不产出。
func TestBuild_TagWhitelistReject(t *testing.T) {
	for _, tag := range []string{"ADD:ONT", "ADD;ONT", "ADD,ONT", "ADD ONT", "B=1", "TRC.1", "中文"} {
		out, err := Build(Command{Verb: "ADD-ONU", Tag: tag}, "B000001")
		if !errors.Is(err, ErrBadParam) {
			t.Fatalf("tag %q: want ErrBadParam got %v", tag, err)
		}
		if out != nil {
			t.Fatalf("tag %q: want no output got %q", tag, out)
		}
	}
}

func TestBuild_WhitelistAllows(t *testing.T) {
	out, err := Build(Command{Verb: "ADD-ONU", Payload: []KV{{"DESC", "PRV-ORD-20260902-0001"}}}, "B2")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := "ADD-ONU:::B2::DESC=PRV-ORD-20260902-0001;"
	if string(out) != want {
		t.Fatalf("got %q want %q", out, want)
	}
}
