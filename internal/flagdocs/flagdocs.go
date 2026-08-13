package flagdocs

import (
	"fmt"
	"strings"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/mongounarchive"
)

func Markdown() string {
	commands := []toolconfig.CommandDoc{
		mongoarchive.FlagDocumentation(),
		mongounarchive.FlagDocumentation(),
	}

	var out strings.Builder
	out.WriteString("# CLI Flags\n\n")
	out.WriteString("This file is generated from the CLI flag definitions. Update the definitions and re-run the verification tests when flags change.\n")
	for i, command := range commands {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString("\n## `")
		out.WriteString(command.Name)
		out.WriteString("`\n\n")
		out.WriteString("| Flag | Environment Variable | Type | Description |\n")
		out.WriteString("| ---- | -------------------- | ---- | ----------- |\n")
		for _, flag := range command.Flags {
			out.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", flag.Flag, flag.EnvVar, flag.Type, escapeCell(flag.Description)))
		}
		if len(command.EnvVars) == 0 {
			continue
		}

		out.WriteString("\n### Environment-Only Variables\n\n")
		out.WriteString("| Environment Variable | Default | Description |\n")
		out.WriteString("| -------------------- | ------- | ----------- |\n")
		for _, envVar := range command.EnvVars {
			defaultValue := envVar.DefaultValue
			if defaultValue == "" {
				defaultValue = "_(none)_"
			}
			out.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", envVar.EnvVar, defaultValue, escapeCell(envVar.Description)))
		}
	}

	return out.String()
}

func escapeCell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
