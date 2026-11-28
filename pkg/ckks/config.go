package ckks

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/filbertengyo/sisterpedia/config"
	ckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	name            string
	logN            int
	logQ            []int
	logP            []int
	logDefaultScale int
	params          ckks.Parameters
}

func (p Profile) Name() string {
	return p.name
}

func (p Profile) LogN() int {
	return p.logN
}

func (p Profile) LogQ() []int {
	return slices.Clone(p.logQ)
}

func (p Profile) LogP() []int {
	return slices.Clone(p.logP)
}

func (p Profile) LogDefaultScale() int {
	return p.logDefaultScale
}

func (p Profile) MarshalBinary() ([]byte, error) {
	return []byte(p.Name()), nil
}

func (p *Profile) UnmarshalBinary(data []byte) error {
	name := string(data)
	profile, err := GetProfile(name)
	if err != nil {
		return err
	}

	p.name = profile.Name()
	p.logN = profile.LogN()
	p.logQ = profile.LogQ()
	p.logP = profile.LogP()
	p.logDefaultScale = profile.LogDefaultScale()
	p.params = profile.params

	return nil
}

func (p Profile) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Name())
}

func (p *Profile) UnmarshalJSON(data []byte) error {
	var name string
	err := json.Unmarshal(data, &name)
	if err != nil {
		return err
	}

	profile, err := GetProfile(name)
	if err != nil {
		return err
	}

	p.name = profile.Name()
	p.logN = profile.LogN()
	p.logQ = profile.LogQ()
	p.logP = profile.LogP()
	p.logDefaultScale = profile.LogDefaultScale()
	p.params = profile.params

	return nil
}

type profileConfig struct {
	LogN            int   `yaml:"log_n"`
	LogQ            []int `yaml:"log_q"`
	LogP            []int `yaml:"log_p"`
	LogDefaultScale int   `yaml:"log_default_scale"`
}

type profilesConfig struct {
	Profiles map[string]profileConfig `yaml:"profiles"`
}

var profiles map[string]Profile

func init() {
	profiles = make(map[string]Profile)

	var profilesConfig profilesConfig
	err := yaml.Unmarshal(config.Profiles, &profilesConfig)
	if err != nil {
		panic(err)
	}

	for name, profile := range profilesConfig.Profiles {
		params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
			LogN:            profile.LogN,
			LogQ:            profile.LogQ,
			LogP:            profile.LogP,
			LogDefaultScale: profile.LogDefaultScale,
		})
		if err != nil {
			panic(err)
		}

		profiles[name] = Profile{
			name:            name,
			logN:            profile.LogN,
			logQ:            profile.LogQ,
			logP:            profile.LogP,
			logDefaultScale: profile.LogDefaultScale,
			params:          params,
		}
	}
}

func GetProfile(name string) (Profile, error) {
	profile, ok := profiles[name]

	if !ok {
		return Profile{}, fmt.Errorf("ckks profile \"%s\" not found", name)
	}

	return profile, nil
}
