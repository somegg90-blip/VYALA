package db

import (
    "embed"
    "os"
    "gopkg.in/yaml.v3"
)

//go:embed *.yaml
var dbFS embed.FS

type PQCStatus struct {
    Status      string `yaml:"status"`
    Algorithm   string `yaml:"algorithm"`
    Replacement string `yaml:"replacement"`
}

// ReadDB loads the PQC readiness database.
func ReadDB() (map[string]map[string]PQCStatus, error) {
    data, err := dbFS.ReadFile("pqc_readiness.yaml")
    if err != nil {
        return nil, err
    }

    var db map[string]map[string]PQCStatus
    if err := yaml.Unmarshal(data, &db); err != nil {
        return nil, err
    }

    return db, nil
}

// Helper to write to temp if needed later, similar to rules.ExtractToTemp
func ExtractToTemp() (string, error) {
    dir, err := os.MkdirTemp("", "pqc-db")
    if err != nil {
        return "", err
    }
    data, err := dbFS.ReadFile("pqc_readiness.yaml")
    if err != nil {
        return "", err
    }
    if err := os.WriteFile(dir+"/pqc_readiness.yaml", data, 0644); err != nil {
        return "", err
    }
    return dir, nil
}
