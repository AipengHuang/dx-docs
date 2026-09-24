package skills

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/sandbox"
)

func TestManagerUsesRemoteSkillSource(t *testing.T) {
	manager := NewManager(&ManagerConfig{
		Enabled: true,
		Remote: &RemoteSkillSource{
			List: func(context.Context) ([]*SkillMetadata, error) {
				return []*SkillMetadata{{Name: "platform-skill", Description: "From platform"}}, nil
			},
			Read: func(_ context.Context, name, path string) (string, []string, error) {
				if name != "platform-skill" || path != SkillFileName {
					t.Fatalf("unexpected remote read: %s/%s", name, path)
				}
				return "---\nname: platform-skill\ndescription: From platform\n---\n\nFollow the platform instructions.", []string{SkillFileName}, nil
			},
		},
	}, sandbox.NewDisabledManager())

	if err := manager.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	metadata := manager.GetAllMetadata()
	if len(metadata) != 1 || metadata[0].Name != "platform-skill" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	skill, err := manager.LoadSkill(context.Background(), "platform-skill")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Instructions != "Follow the platform instructions." {
		t.Fatalf("unexpected instructions: %q", skill.Instructions)
	}
	files, err := manager.ListSkillFiles(context.Background(), "platform-skill")
	if err != nil || len(files) != 1 || files[0] != SkillFileName {
		t.Fatalf("unexpected files: %#v, %v", files, err)
	}
}
