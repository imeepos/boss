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
