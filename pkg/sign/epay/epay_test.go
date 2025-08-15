package epay

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSign(t *testing.T) {
	testCases := []struct {
		name      string
		apiKey    string
		data      map[string]any
		wantSign  string
		wantError error
	}{
		{
			name:   "正常生成",
			apiKey: "37a96739aece637fb567fd40e6526116",
			data: map[string]any{
				"epayAccount":        "jing32@qq.com",
				"transactionType":    "C2C",
				"category":           "BANK",
				"notifyUrl":          "http://192.168.2.6:8380/openapi/apiForward/callBack",
				"merchantOrderNo":    "202201019002",
				"amount":             "100",
				"receiveAmount":      "",
				"settlementCurrency": "USD",
				"receiveCurrency":    "CNY",
				"version":            "V2.0.0",
				"senderInfo": map[string]any{
					"surName":             "Joe",
					"givName":             "Chang",
					"idNumber":            "A199267867",
					"idType":              "1",
					"birthday":            "1986-09-11",
					"country":             "US",
					"address":             "Santa Clara, CA 3050 Bowers Avenue",
					"purposeOfRemittance": "20",
					"city":                "Santa Clara",
					"zipCode":             "58039",
					"accountNo":           "13721473389",
				},
				"receiverInfo": map[string]any{
					"surName":     "lu",
					"givName":     "hui",
					"nationality": "CN",
					"country":     "CN",
					"area":        "",
					"accountNo":   "13721473389",
					"idNumber":    "1234567890",
				},
			},
			wantSign:  "20AF5C801E96D105D5354AF2A3C8BB0423092F2F2B6D631412C9742047C2EF08",
			wantError: nil,
		},
		{
			name:   "apiKey 为空",
			apiKey: "",
			data: map[string]any{
				"epayAccount":        "jing32@qq.com",
				"transactionType":    "C2C",
				"category":           "BANK",
				"notifyUrl":          "http://192.168.2.6:8380/openapi/apiForward/callBack",
				"merchantOrderNo":    "202201019002",
				"amount":             "100",
				"receiveAmount":      "",
				"settlementCurrency": "USD",
				"receiveCurrency":    "CNY",
				"version":            "V2.0.0",
				"senderInfo": map[string]any{
					"surName":             "Joe",
					"givName":             "Chang",
					"idNumber":            "A199267867",
					"idType":              "1",
					"birthday":            "1986-09-11",
					"country":             "US",
					"address":             "Santa Clara, CA 3050 Bowers Avenue",
					"purposeOfRemittance": "20",
					"city":                "Santa Clara",
					"zipCode":             "58039",
					"accountNo":           "13721473389",
				},
				"receiverInfo": map[string]any{
					"surName":     "lu",
					"givName":     "hui",
					"nationality": "CN",
					"country":     "CN",
					"area":        "",
					"accountNo":   "13721473389",
					"idNumber":    "1234567890",
				},
			},
			wantSign:  "",
			wantError: errors.New("apiKey cannot be empty"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signer := NewEpaySignHandler()
			sign, err := signer.GenerateSign(tc.data, tc.apiKey)
			assert.Equal(t, tc.wantSign, sign)
			assert.Equal(t, tc.wantError, err)
		})
	}
}
