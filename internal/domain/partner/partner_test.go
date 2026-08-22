// 入驻域纯逻辑单测(无 DB):申请校验 + 口令生成。
package partner

import (
	"strings"
	"testing"
)

func validApplication() Application {
	return Application{
		CompanyName: "ACME Telecom", CreditCode: "US12345678",
		ContactName: "John", ContactPhone: "+639170000001",
		BusinessDesc: "宽带转售合作",
	}
}

func TestApplicationValidate(t *testing.T) {
	if err := validApplication().Validate(); err != nil {
		t.Fatalf("valid app rejected: %v", err)
	}
	for _, field := range []string{"companyName", "creditCode", "contactName", "contactPhone", "businessDesc"} {
		a := validApplication()
		switch field {
		case "companyName":
			a.CompanyName = " "
		case "creditCode":
			a.CreditCode = ""
		case "contactName":
			a.ContactName = " "
		case "contactPhone":
			a.ContactPhone = ""
		case "businessDesc":
			a.BusinessDesc = " "
		}
		if err := a.Validate(); err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("%s: want required error, got %v", field, err)
		}
	}
}

func TestGenPassword(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		pwd, err := genPassword()
		if err != nil {
			t.Fatalf("genPassword: %v", err)
		}
		if len(pwd) != 12 {
			t.Fatalf("length = %d, want 12", len(pwd))
		}
		if seen[pwd] {
			t.Fatalf("duplicate password generated: %s", pwd)
		}
		seen[pwd] = true
	}
}
