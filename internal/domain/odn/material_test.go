package odn

import (
	"errors"
	"testing"
)

func TestValidateIssueTransition(t *testing.T) {
	oks := []struct{ from, to string }{
		{IssueOpen, IssueConfirmed}, {IssueOpen, IssueCancelled},
		{IssueConfirmed, IssueCancelled}, {IssueOpen, IssueOpen},
	}
	for _, c := range oks {
		if err := ValidateIssueTransition(c.from, c.to); err != nil {
			t.Fatalf("%s->%s want ok, got %v", c.from, c.to, err)
		}
	}
	bad := []struct{ from, to string }{
		{IssueConfirmed, IssueConfirmed + "X"}, {IssueCancelled, IssueOpen},
		{IssueCancelled, IssueConfirmed}, {IssueConfirmed, IssueOpen},
	}
	for _, c := range bad {
		if err := ValidateIssueTransition(c.from, c.to); !errors.Is(err, ErrIssueState) {
			t.Fatalf("%s->%s want ErrIssueState, got %v", c.from, c.to, err)
		}
	}
}
