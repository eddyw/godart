package sdk

import (
	_ "embed"
	"encoding/json"
)

//go:embed compatibility.json
var manifest []byte

type Versions struct {
	Go struct {
		Baseline   string   `json:"baseline"`
		Candidates []string `json:"candidates"`
	} `json:"go"`
	Dart struct {
		Baseline           string   `json:"baseline"`
		Resident           []string `json:"resident"`
		SourceMin          string   `json:"source_min"`
		SourceMaxExclusive string   `json:"source_max_exclusive"`
	} `json:"dart"`
	Flutter struct {
		Development string `json:"development"`
		Dart        string `json:"dart"`
	} `json:"flutter"`
}

func Read() (Versions, error) {
	var versions Versions
	err := json.Unmarshal(manifest, &versions)
	return versions, err
}
