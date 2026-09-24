package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCreateSkillToolWritesOneCompleteSkill(t *testing.T) {
	var name, content string
	tool := NewCreateSkillTool(func(_ context.Context, skillName, skillContent string) error {
		name, content = skillName, skillContent
		return nil
	})

	result, err := tool.Execute(context.Background(), json.RawMessage(`{
		"name":"contract-review",
		"content":"---\nname: contract-review\ndescription: review contracts\n---\n\nCheck the contract and report risks."
	}`))
	if err != nil || !result.Success {
		t.Fatalf("create skill failed: result=%+v err=%v", result, err)
	}
	if name != "contract-review" || content == "" {
		t.Fatalf("unexpected write: name=%q content=%q", name, content)
	}
}
