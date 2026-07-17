package agency

import (
	"strings"
	"testing"
)

func validInput() applicationInput {
	return applicationInput{
		Name:         "张三",
		Phone:        "13800138000",
		Region:       "北京市 北京市 朝阳区",
		ChannelType:  "community",
		AudienceSize: "500_2000",
		Experience:   "运营本地创业者社群三年，可以通过社群活动推广。",
		Consent:      true,
	}
}

func TestValidateInput(t *testing.T) {
	if err := validateInput(validInput()); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}

	tests := []struct {
		name string
		edit func(*applicationInput)
		code string
	}{
		{"phone", func(in *applicationInput) { in.Phone = "123" }, "INVALID_PHONE"},
		{"channel", func(in *applicationInput) { in.ChannelType = "unknown" }, "INVALID_CHANNEL_TYPE"},
		{"experience", func(in *applicationInput) { in.Experience = "太短" }, "INVALID_EXPERIENCE"},
		{"consent", func(in *applicationInput) { in.Consent = false }, "CONSENT_REQUIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.edit(&in)
			err := validateInput(in)
			if err == nil || err.Code != tt.code {
				t.Fatalf("got %#v, want code %s", err, tt.code)
			}
		})
	}
}

func TestAgencyCodePattern(t *testing.T) {
	for _, code := range []string{"1112", "2048", "9999"} {
		if !agencyCodePattern.MatchString(code) {
			t.Fatalf("valid agency code rejected: %q", code)
		}
	}
	for _, code := range []string{"111", "10000", "WF12345678", "12A4"} {
		if agencyCodePattern.MatchString(code) {
			t.Fatalf("invalid agency code accepted: %q", code)
		}
	}
}

func TestMaskedUserDoesNotExposeFullID(t *testing.T) {
	fullID := "user-sensitive-123456789"
	masked := maskedUser(fullID)
	if strings.Contains(masked, fullID) || masked != "用户 456789" {
		t.Fatalf("unexpected masked id: %q", masked)
	}
}
