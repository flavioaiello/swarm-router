package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type conf struct {
	Env      map[string]string
	Mappings map[string]string
}

func envMap() map[string]string {
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		x := strings.SplitN(kv, "=", 2)
		m[x[0]] = x[1]
	}
	return m
}

func backendMap() map[string]string {
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "BE_") {
			x := strings.SplitN(kv[3:], "=", 2)
			m[x[0]] = x[1]
		}
	}
	return m
}

func verify(mapping string) bool {
	for _, backendVerifyTLS := range strings.Split(backendsVerifyTLS, " ") {
		if backendVerifyTLS == mapping {
			return true
		}
	}
	return false
}

func newTemplate(name string) *template.Template {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"split":  strings.Split,
		"splitN": strings.SplitN,
		"verify": verify,
	})
	return tmpl
}

func executeTemplate(tmpl string, cfg string) {
	config := new(conf)
	config.Env = envMap()
	config.Mappings = backendMap()

	template, err := newTemplate(filepath.Base(tmpl)).ParseFiles(tmpl)
	if err != nil {
		log.Fatalf("Unable to parse template error: %s", err.Error())
		return
	}
	file, err := os.Create(cfg)
	if err != nil {
		log.Printf("Create file error: %s", err.Error())
		return
	}
	err = template.Execute(file, config)
	if err != nil {
		log.Printf("Execute template error: %s", err.Error())
		return
	}
	file.Close()
}
