package main

import (
  "os"
  "text/template"
  "path/filepath"
  "log"
  "strings"
  "strconv"
)

type Config struct {
  HttpSwarmRouterPort int
  TlsSwarmRouterPort int
  Env map[string]string
  HttpBackends map[string]int
  TlsBackends map[string]int
}

func envMap() map[string]string {
  m := make(map[string]string)
  for _, kv := range os.Environ() {
    x := strings.SplitN(kv, "=", 2)
    m[x[0]] = x[1]
  }
  return m
}

func getBackend(hostname string) string {
  if !dnsBackendFqdn {
    hostname = strings.Split(hostname, ".")[0]
  }
  if dnsBackendSuffix != "" {
    hostname = hostname + dnsBackendSuffix
  }
  return hostname
}

func newTemplate(name string) *template.Template {
  tmpl := template.New(name).Funcs(template.FuncMap{
    "split": strings.Split,
    "splitN": strings.SplitN,
    "getBackend": getBackend,
  })
  return tmpl
}

func executeTemplate(tmpl string, cfg string) {
  config := new(Config)
  config.HttpSwarmRouterPort, _ = strconv.Atoi(httpSwarmRouterPort)
  config.TlsSwarmRouterPort, _ = strconv.Atoi(tlsSwarmRouterPort)
  config.Env = envMap()
  config.HttpBackends = httpBackends
  config.TlsBackends = tlsBackends

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
