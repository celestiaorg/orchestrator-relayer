package common

import (
	"fmt"
	"strings"
)

func CommandToServiceName(commandUsage string) (string, error) {
	if strings.Contains(commandUsage, "rel") {
		return "relayer", nil
	}
	if strings.Contains(commandUsage, "orch") {
		return "orchestrator", nil
	}
	if strings.Contains(commandUsage, "deploy") {
		return "deployer", nil
	}
	return "", fmt.Errorf("unknown service %s", commandUsage)
}
