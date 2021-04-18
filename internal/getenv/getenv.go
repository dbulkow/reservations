/* Copyright (c) 2021 David Bulkow */

package getenv

import (
	"os"
	"strconv"
	"strings"
)

type Env struct {
	prefix string
}

func NewEnv(prefix string) *Env {
	return &Env{prefix: prefix}
}

func (e *Env) varname(suffix string) string {
	if e.prefix == "" {
		return suffix
	}
	return strings.Join([]string{e.prefix, suffix}, "_")
}

func (e *Env) Get(suffix, defvalue string) string {
	env := os.Getenv(e.varname(suffix))

	if env == "" {
		return defvalue
	}

	return env
}

func (e *Env) GetInt(suffix string, defvalue int) int {
	env := os.Getenv(e.varname(suffix))

	if env == "" {
		return defvalue
	}

	v, err := strconv.Atoi(env)
	if err != nil {
		return defvalue
	}

	return int(v)
}

func (e *Env) GetBool(suffix string, defvalue bool) bool {
	env := os.Getenv(e.varname(suffix))

	if env == "" {
		return defvalue
	}

	if strings.ToLower(env) == "true" {
		return true
	}

	return false
}

var defEnv = &Env{}

func GetEnv(varname, defvalue string) string {
	return defEnv.Get(varname, defvalue)
}

func GetEnvInt(varname string, defvalue int) int {
	return defEnv.GetInt(varname, defvalue)
}

func GetEnvBool(varname string, defvalue bool) bool {
	return defEnv.GetBool(varname, defvalue)
}
