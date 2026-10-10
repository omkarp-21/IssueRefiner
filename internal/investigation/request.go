package investigation

import (
	"strings"
)

const MaxInformationRequestBytes = 6000

func sanitizeMention(item string) string {
	return strings.ReplaceAll(item, "@", "@\u200b")
}

func BuildInformationRequest(missing []string) string {
	var draft strings.Builder
	draft.WriteString("To help investigate this issue, could you please provide the following information?\n")
	for _, item := range missing {
		if item = strings.TrimSpace(item); item != "" {
			draft.WriteString("\n- " + sanitizeMention(item))
		}
	}
	draft.WriteString("\n\nOnce this information is available, the issue can be investigated further.\n\n_Posted via Issue Refiner._")
	return truncateTo(draft.String(), MaxInformationRequestBytes)
}
