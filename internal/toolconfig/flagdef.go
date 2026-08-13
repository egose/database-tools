package toolconfig

import "strings"

type EnvReader interface {
	GetValue(string, ...string) string
}

type FlagBinder interface {
	String(string, string, string) *string
	Bool(string, bool, string) *bool
}

type FlagDoc struct {
	Flag        string
	EnvVar      string
	Type        string
	Description string
}

type EnvDoc struct {
	EnvVar       string
	DefaultValue string
	Description  string
}

type CommandDoc struct {
	Name    string
	Flags   []FlagDoc
	EnvVars []EnvDoc
}

type StringFlagDef struct {
	Name        string
	EnvKey      string
	Usage       string
	Defaults    []string
	TypeName    string
	DocEnvVar   string
	DocFlagName string
}

func (d StringFlagDef) Bind(fs FlagBinder, env EnvReader) *string {
	return fs.String(d.Name, envValue(env, d.EnvKey, d.Defaults...), d.Usage)
}

func (d StringFlagDef) Doc(envPrefix string) FlagDoc {
	flagName := d.DocFlagName
	if flagName == "" {
		flagName = "`--" + d.Name + "`"
	}
	envVar := d.DocEnvVar
	if envVar == "" && d.EnvKey != "" {
		envVar = "`" + envPrefix + d.EnvKey + "`"
	}
	typeName := d.TypeName
	if typeName == "" {
		typeName = "string"
	}
	if envVar == "" {
		envVar = "_(no env var)_"
	}
	return FlagDoc{Flag: flagName, EnvVar: envVar, Type: typeName, Description: d.Usage}
}

type BoolFlagDef struct {
	Name        string
	EnvKey      string
	Usage       string
	Default     bool
	DocEnvVar   string
	DocFlagName string
}

func (d BoolFlagDef) Bind(fs FlagBinder, env EnvReader) *bool {
	return fs.Bool(d.Name, envBool(env, d.EnvKey, d.Default), d.Usage)
}

func (d BoolFlagDef) Doc(envPrefix string) FlagDoc {
	flagName := d.DocFlagName
	if flagName == "" {
		flagName = "`--" + d.Name + "`"
	}
	envVar := d.DocEnvVar
	if envVar == "" && d.EnvKey != "" {
		envVar = "`" + envPrefix + d.EnvKey + "`"
	}
	if envVar == "" {
		envVar = "_(no env var)_"
	}
	return FlagDoc{Flag: flagName, EnvVar: envVar, Type: "bool", Description: d.Usage}
}

func envValue(env EnvReader, key string, defaults ...string) string {
	if env == nil || key == "" {
		for _, fallback := range defaults {
			if fallback != "" {
				return fallback
			}
		}
		return ""
	}

	return env.GetValue(key, defaults...)
}

func envBool(env EnvReader, key string, fallback bool) bool {
	if env == nil || key == "" {
		return fallback
	}

	return strings.EqualFold(env.GetValue(key), "true")
}
