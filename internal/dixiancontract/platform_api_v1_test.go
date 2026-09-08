package dixiancontract

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestGeneratedGoContractMatchesSyncedOpenAPI(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "contracts", "dixian", "platform-api-v1.json"))
	if err != nil {
		t.Fatal(err)
	}

	var contract struct {
		Components struct {
			Schemas struct {
				IdentitySource struct {
					Enum []string `json:"enum"`
				} `json:"IdentitySource"`
				ScopeKind struct {
					Enum []string `json:"enum"`
				} `json:"ScopeKind"`
				DeviceChannel struct {
					Enum []string `json:"enum"`
				} `json:"DeviceChannel"`
				RequestContext struct {
					Required   []string                   `json:"required"`
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"RequestContext"`
				SSEEventType struct {
					Enum []string `json:"enum"`
				} `json:"SseEventType"`
				ErrorCode struct {
					HTTPStatus map[string]int `json:"x-dixian-http-status"`
				} `json:"ErrorCode"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatal(err)
	}

	if got := fmt.Sprintf("%x", sha256.Sum256(content)); ContractSHA256 != got {
		t.Fatalf("contract hash drifted: %s", ContractSHA256)
	}
	propertyNames := make([]string, 0, len(contract.Components.Schemas.RequestContext.Properties))
	for name := range contract.Components.Schemas.RequestContext.Properties {
		propertyNames = append(propertyNames, name)
	}
	sort.Strings(propertyNames)
	wantPropertyNames := append([]string(nil), RequestContextFields...)
	sort.Strings(wantPropertyNames)
	if !reflect.DeepEqual(wantPropertyNames, propertyNames) {
		t.Fatalf("request context properties drifted: %v", RequestContextFields)
	}
	assertStructFields(t, RequestContext{}, RequestContextFields)
	assertStructFields(t, ErrorEnvelope{}, ErrorEnvelopeFields)
	assertStructFields(t, CursorPage[any]{}, CursorPageFields)
	assertStructFields(t, SSERunEvent{}, SSERunEventFields)
	if !reflect.DeepEqual(RequestContextRequiredFields, contract.Components.Schemas.RequestContext.Required) {
		t.Fatalf("request context fields drifted: %v", RequestContextRequiredFields)
	}
	if !reflect.DeepEqual(SSEEventTypes, contract.Components.Schemas.SSEEventType.Enum) {
		t.Fatalf("SSE event types drifted: %v", SSEEventTypes)
	}
	if !reflect.DeepEqual(IdentitySources, contract.Components.Schemas.IdentitySource.Enum) {
		t.Fatalf("identity sources drifted: %v", IdentitySources)
	}
	if !reflect.DeepEqual(ScopeKinds, contract.Components.Schemas.ScopeKind.Enum) {
		t.Fatalf("scope kinds drifted: %v", ScopeKinds)
	}
	if !reflect.DeepEqual(DeviceChannels, contract.Components.Schemas.DeviceChannel.Enum) {
		t.Fatalf("device channels drifted: %v", DeviceChannels)
	}
	if !reflect.DeepEqual(ErrorHTTPStatus, contract.Components.Schemas.ErrorCode.HTTPStatus) {
		t.Fatalf("error status mapping drifted: %v", ErrorHTTPStatus)
	}
	if !reflect.DeepEqual(SSETerminalEventTypes, []string{"completed", "failed", "cancelled"}) {
		t.Fatalf("terminal SSE event types drifted: %v", SSETerminalEventTypes)
	}
}

func assertStructFields(t *testing.T, value any, expected []string) {
	t.Helper()
	typeOf := reflect.TypeOf(value)
	actual := make([]string, 0, typeOf.NumField())
	for index := 0; index < typeOf.NumField(); index++ {
		actual = append(actual, strings.Split(typeOf.Field(index).Tag.Get("json"), ",")[0])
	}
	sort.Strings(actual)
	want := append([]string(nil), expected...)
	sort.Strings(want)
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("%s fields drifted: got %v want %v", typeOf.Name(), actual, want)
	}
}
