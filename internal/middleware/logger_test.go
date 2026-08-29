package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDPropagatesLogNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "request-1")
	request.Header.Set("X-Log-Number", "LOG-01ARZ3NDEKTSV4RRFFQ69G5FAV")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got != "request-1" {
		t.Fatalf("X-Request-ID = %q", got)
	}
	if got := response.Header().Get("X-Log-Number"); got != "LOG-01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("X-Log-Number = %q", got)
	}
}

func TestParseLogNumber(t *testing.T) {
	cases := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "valid", value: "LOG-01ARZ3NDEKTSV4RRFFQ69G5FAV", valid: true},
		{name: "wrong prefix", value: "JOB-01ARZ3NDEKTSV4RRFFQ69G5FAV", valid: false},
		{name: "wrong length", value: "LOG-01ARZ3NDEKTSV4RRFFQ69G5FA", valid: false},
		{name: "invalid first digit", value: "LOG-81ARZ3NDEKTSV4RRFFQ69G5FAV", valid: false},
		{name: "ambiguous character", value: "LOG-01ARZ3NDEKTSV4RRFFQ69G5FAI", valid: false},
		{name: "lowercase", value: "LOG-01ARZ3NDEKTSV4RRFFQ69G5Fav", valid: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			value, valid := parseLogNumber(testCase.value)
			if valid != testCase.valid {
				t.Fatalf("parseLogNumber(%q) valid = %v", testCase.value, valid)
			}
			if valid && value != testCase.value {
				t.Fatalf("parseLogNumber(%q) = %q", testCase.value, value)
			}
		})
	}
}

func TestSanitizeBody(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "camelCase apiKey",
			in:   `{"modelName":"gpt-5.2","apiKey":"sk-secret-123","provider":"azure_openai"}`,
			want: `{"modelName":"gpt-5.2","apiKey":"***","provider":"azure_openai"}`,
		},
		{
			name: "snake_case api_key",
			in:   `{"api_key":"sk-secret-123"}`,
			want: `{"api_key":"***"}`,
		},
		{
			name: "PascalCase APIKey",
			in:   `{"APIKey":"sk-secret-123"}`,
			want: `{"APIKey":"***"}`,
		},
		{
			name: "secretKey camelCase",
			in:   `{"secretKey":"abc","accessKeyId":"id"}`,
			want: `{"secretKey":"***","accessKeyId":"id"}`,
		},
		{
			name: "refreshToken / accessToken camelCase",
			in:   `{"refreshToken":"rt","accessToken":"at"}`,
			want: `{"refreshToken":"***","accessToken":"***"}`,
		},
		{
			name: "password and token preserved as masked",
			in:   `{"password":"p","token":"t"}`,
			want: `{"password":"***","token":"***"}`,
		},
		{
			name: "snake_case new_password and old_password",
			in:   `{"email":"alice@example.com","new_password":"FreshPass9","old_password":"OldPass9"}`,
			want: `{"email":"alice@example.com","new_password":"***","old_password":"***"}`,
		},
		{
			name: "extra whitespace around colon",
			in:   `{"apiKey"  :   "leak"}`,
			want: `{"apiKey":"***"}`,
		},
		{
			name: "non sensitive fields untouched",
			in:   `{"baseUrl":"https://example.com","modelName":"gpt"}`,
			want: `{"baseUrl":"https://example.com","modelName":"gpt"}`,
		},
		{
			name: "OAuth authorization response fields",
			in:   `{"authorization_url":"https://idp.example/authorize?state=secret","authorization_attempt":"secret-state"}`,
			want: `{"authorization_url":"***","authorization_attempt":"***"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeBody(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeBody(%q)\n got: %s\nwant: %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeQuery(t *testing.T) {
	got := sanitizeQuery("code=secret-code&state=secret-state&next=%2Fsettings&state=second")
	want := "code=%2A%2A%2A&next=%2Fsettings&state=%2A%2A%2A"
	if got != want {
		t.Fatalf("sanitizeQuery() = %q, want %q", got, want)
	}
}
