package agency

import "testing"

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
