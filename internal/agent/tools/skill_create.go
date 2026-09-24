package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

const maxSkillContentBytes = 200_000

var createSkillTool = BaseTool{
	name: ToolCreateSkill,
	description: `Create one reusable Skill after the user explicitly asks for it.

Ask only for missing purpose, scenarios, and steps. Then write one complete SKILL.md with valid YAML frontmatter containing name and description, followed by clear instructions.`,
	schema: utils.GenerateSchema[CreateSkillInput](),
}

type CreateSkillInput struct {
	Name    string `json:"name" jsonschema:"Skill name from the SKILL.md frontmatter"`
	Content string `json:"content" jsonschema:"Complete SKILL.md content including YAML frontmatter and instructions"`
}

type CreateSkillTool struct {
	BaseTool
	write func(context.Context, string, string) error
}

func NewCreateSkillTool(write func(context.Context, string, string) error) *CreateSkillTool {
	return &CreateSkillTool{BaseTool: createSkillTool, write: write}
}

func (t *CreateSkillTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var input CreateSkillInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("Invalid skill: %v", err)}, nil
	}
	if t.write == nil || input.Name == "" || input.Content == "" || len(input.Content) > maxSkillContentBytes {
		return &types.ToolResult{Success: false, Error: "Skill name and content are required"}, nil
	}
	parsed, err := skills.ParseSkillFile(input.Content)
	if err != nil || parsed.Name != input.Name || parsed.Instructions == "" {
		return &types.ToolResult{Success: false, Error: "Skill content must contain matching name, description, and instructions"}, nil
	}
	if err := t.write(ctx, input.Name, input.Content); err != nil {
		return &types.ToolResult{Success: false, Error: "Skill could not be saved"}, nil
	}
	return &types.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Skill %q was created and is ready to use.", input.Name),
		Data:    map[string]interface{}{"name": input.Name},
	}, nil
}

func (t *CreateSkillTool) Cleanup(context.Context) error { return nil }
