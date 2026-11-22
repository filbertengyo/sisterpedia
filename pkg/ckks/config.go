package ckks

import (
	"errors"

	"github.com/filbertengyo/sisterpedia/config"
	ckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	LogN            int   `yaml:"log_n"`
	LogQ            []int `yaml:"log_q"`
	LogP            []int `yaml:"log_p"`
	LogDefaultScale int   `yaml:"log_default_scale"`
}

func (p Profile) asLattigoParams() ckks.ParametersLiteral {
	return ckks.ParametersLiteral{
		LogN:            p.LogN,
		LogQ:            p.LogQ,
		LogP:            p.LogP,
		LogDefaultScale: p.LogDefaultScale,
	}
}

type profilesConfig struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

var profiles profilesConfig

func init() {
	err := yaml.Unmarshal(config.Profiles, &profiles)
	if err != nil {
		panic(err)
	}
}

func GetProfile(p string) (*Profile, error) {
	profile, ok := profiles.Profiles[p]

	if !ok {
		return nil, errors.New("ckks profile not found")
	}

	return &profile, nil
}
