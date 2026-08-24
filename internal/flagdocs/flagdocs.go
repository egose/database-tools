package flagdocs

import (
	"fmt"
	"strings"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/mongounarchive"
	"github.com/egose/database-tools/postgresarchive"
	"github.com/egose/database-tools/postgresunarchive"
)

func Markdown() string {
	commands := []toolconfig.CommandDoc{
		mongoarchive.FlagDocumentation(),
		mongounarchive.FlagDocumentation(),
		postgresarchive.FlagDocumentation(),
		postgresunarchive.FlagDocumentation(),
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
		writePostgreSQLFallbackEnvDocs(&out, command)
	}

	return out.String()
}

func writePostgreSQLFallbackEnvDocs(out *strings.Builder, command toolconfig.CommandDoc) {
	commandPrefix := ""
	switch command.Name {
	case "postgres-archive":
		commandPrefix = "POSTGRESARCHIVE__"
	case "postgres-unarchive":
		commandPrefix = "POSTGRESUNARCHIVE__"
	default:
		return
	}

	keys := make([]string, 0, len(command.Flags)+len(command.EnvVars))
	seen := map[string]bool{}
	add := func(markdownEnv string) {
		if !strings.HasPrefix(markdownEnv, "`") || !strings.HasSuffix(markdownEnv, "`") {
			return
		}
		envVar := strings.Trim(markdownEnv, "`")
		key, ok := strings.CutPrefix(envVar, commandPrefix)
		if !ok || seen[key] {
			return
		}
		seen[key] = true
		keys = append(keys, key)
	}
	for _, flag := range command.Flags {
		add(flag.EnvVar)
	}
	for _, envVar := range command.EnvVars {
		add("`" + envVar.EnvVar + "`")
	}
	if len(keys) == 0 {
		return
	}

	out.WriteString("\n### PostgreSQL Environment Fallbacks\n\n")
	out.WriteString("PostgreSQL commands read command-specific variables first, then shared PostgreSQL variables, then unprefixed variables.\n\n")
	out.WriteString("| Key | Lookup Order |\n")
	out.WriteString("| --- | ------------ |\n")
	for _, key := range keys {
		out.WriteString(fmt.Sprintf("| `%s` | `%s%s`, `POSTGRES__%s`, `%s` |\n", key, commandPrefix, key, key, key))
	}
}

func escapeCell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
