package main

import (
  "os"
	"text/template"
	"path/filepath"
	"log"
  "strings"
)

type Config struct {
	Env map[string]string
  Backends map[string]string
}

func envMap() map[string]string {
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		x := strings.SplitN(kv, "=", 2)
		m[x[0]] = x[1]
	}
	return m
}

func newTemplate(name string) *template.Template {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"split": strings.Split,
		"splitN": strings.SplitN,
	})
	return tmpl
}

func executeTemplate(tmpl string, cfg string) {
  config := new(Config)
  config.Env = envMap()
  config.Backends = backends
  
  template, err := newTemplate(filepath.Base(tmpl)).ParseFiles(tmpl)
  if err != nil {
  	log.Fatalf("Unable to parse template: %s", err)
    return
  }
	file, err := os.Create(cfg)
	if err != nil {
		log.Println("create file: ", err)
		return
	}
	err = template.Execute(file, config)
	if err != nil {
		log.Print("execute: ", err)
		return
	}
	file.Close()
}
