package shared

import (
	"fmt"
	"strings"

	"aeonechoes/server/internal/domain"
)

func BuildOptimizedPrompt(seed domain.ProjectSeed) string {
	parts := []string{
		fmt.Sprintf("标题：%s", seed.Title),
		fmt.Sprintf("核心设定：%s", seed.Premise),
		fmt.Sprintf("类型 / 语气 / 读者：%s / %s / %s", FirstNonEmpty(seed.Genre, "未分类"), FirstNonEmpty(seed.Tone, "稳健、清晰"), FirstNonEmpty(seed.Audience, "通用读者")),
		fmt.Sprintf("舞台：%s", FirstNonEmpty(seed.Setting, "待扩展")),
	}
	if len(seed.Themes) > 0 {
		parts = append(parts, "主题："+strings.Join(seed.Themes, "、"))
	}
	if len(seed.MainCharacters) > 0 {
		parts = append(parts, "关键角色："+strings.Join(seed.MainCharacters, "、"))
	}
	if len(seed.Constraints) > 0 {
		parts = append(parts, "约束："+strings.Join(seed.Constraints, "；"))
	}
	return strings.Join(parts, "\n")
}
