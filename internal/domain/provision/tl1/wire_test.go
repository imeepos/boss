package tl1

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadFrame_Single(t *testing.T) {
	msg := "   HW_10.71.227.225 2010-07-27 14:32:06\nM  1 COMPLD\n   EN=0   ENDESC=ok\n"
	got, err := ReadFrame(strings.NewReader(msg + ";"))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if got != msg {
		t.Fatalf("got %q want %q", got, msg)
	}
}

func TestReadFrame_MultiBlock(t *testing.T) {
	block1 := "...data..."
	block2 := "...more..."
	got, err := ReadFrame(strings.NewReader(block1 + ">" + block2 + ";"))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	want := block1 + block2
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReadFrame_EOF(t *testing.T) {
	_, err := ReadFrame(strings.NewReader(""))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("want io.EOF got %v", err)
	}
}

func TestReadFrame_TrailingGtEOF(t *testing.T) {
	_, err := ReadFrame(strings.NewReader("abc>"))
	if !errors.Is(err, ErrConnBroken) {
		t.Fatalf("want ErrConnBroken got %v", err)
	}
}
