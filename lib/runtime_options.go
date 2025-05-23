package lib

import (
	"fmt"
	"strings"

	"gopkg.in/guregu/null.v3"
)

// CompatibilityMode specifies the JS compatibility mode
//
//go:generate enumer -type=CompatibilityMode -transform=snake -trimprefix CompatibilityMode -output compatibility_mode_gen.go
type CompatibilityMode uint8

const (
	// CompatibilityModeExtended adds `global` as an alias for `globalThis` on top of the base mode
	CompatibilityModeExtended CompatibilityMode = iota + 1
	// CompatibilityModeBase is standard Sobek, which means pure vanilla JS following ECMAScript standards.
	CompatibilityModeBase
	// CompatibilityModeExperimentalEnhanced achieves TypeScript and ES6+ compatibility with esbuild
	CompatibilityModeExperimentalEnhanced
)

// RuntimeOptions are settings passed onto the Sobek JS runtime
type RuntimeOptions struct {
	TestType null.String `json:"-"`

	// Whether to pass the actual system environment variables to the JS runtime
	IncludeSystemEnvVars null.Bool `json:"includeSystemEnvVars"`

	// JS compatibility mode: "extended" (Sobek+global) or "base" (plain Sobek)
	//
	// TODO: when we resolve https://github.com/k6io/k6/issues/883, we probably
	// should use the CompatibilityMode type directly... but by then, we'd need to have
	// some way of knowing if the value has been set by the user or if we're using the
	// default one, so we can handle `k6 run --compatibility-mode=base es6_extended_archive.tar`
	CompatibilityMode null.String `json:"compatibilityMode"`

	// Environment variables passed onto the runner
	Env map[string]string `json:"env"`

	NoThresholds  null.Bool   `json:"noThresholds"`
	NoSummary     null.Bool   `json:"noSummary"`
	SummaryMode   null.String `json:"summaryMode"`
	SummaryExport null.String `json:"summaryExport"`
	KeyWriter     null.String `json:"-"`
	TracesOutput  null.String `json:"tracesOutput"`
}

// ValidateCompatibilityMode checks if the provided val is a valid compatibility mode
func ValidateCompatibilityMode(val string) (cm CompatibilityMode, err error) {
	if val == "" {
		return CompatibilityModeExtended, nil
	}
	if cm, err = CompatibilityModeString(val); err != nil {
		var compatValues []string
		for _, v := range CompatibilityModeValues() {
			compatValues = append(compatValues, v.String())
		}
		err = fmt.Errorf(`invalid compatibility mode "%s". Use: "%s"`,
			val, strings.Join(compatValues, `", "`))
	}
	return
}
