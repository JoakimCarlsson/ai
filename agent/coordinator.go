package agent

import "github.com/joakimcarlsson/ai/tool"

var teamToolNames = map[string]struct{}{
	"spawn_teammate":      {},
	"stop_teammate":       {},
	"send_message":        {},
	"read_messages":       {},
	"list_teammates":      {},
	"create_board_task":   {},
	"claim_board_task":    {},
	"complete_board_task": {},
	"list_board_tasks":    {},
}

func filterToTeamTools(tools []tool.BaseTool) []tool.BaseTool {
	filtered := make([]tool.BaseTool, 0, len(tools))
	for _, t := range tools {
		if _, ok := teamToolNames[t.Info().Name]; ok {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
