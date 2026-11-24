package ckks

import (
	"encoding/json"
	"fmt"

	"github.com/filbertengyo/sisterpedia/config"
	ckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name            string `yaml:"-"`
	LogN            int    `yaml:"log_n"`
	LogQ            []int  `yaml:"log_q"`
	LogP            []int  `yaml:"log_p"`
	LogDefaultScale int    `yaml:"log_default_scale"`
	params          ckks.Parameters
}

func (p Profile) MarshalBinary() ([]byte, error) {
	return []byte(p.Name), nil
}

func (p *Profile) UnmarshalBinary(data []byte) error {
	name := string(data)
	profile, err := GetProfile(name)
	if err != nil {
		return err
	}

	p.Name = profile.Name
	p.LogN = profile.LogN
	p.LogQ = profile.LogQ
	p.LogP = profile.LogP
	p.LogDefaultScale = profile.LogDefaultScale
	p.params = profile.params

	return nil
}

func (p Profile) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Name)
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

	p.Name = profile.Name
	p.LogN = profile.LogN
	p.LogQ = profile.LogQ
	p.LogP = profile.LogP
	p.LogDefaultScale = profile.LogDefaultScale
	p.params = profile.params

	return nil
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

	for name, profile := range profiles.Profiles {
		params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
			LogN:            profile.LogN,
			LogQ:            profile.LogQ,
			LogP:            profile.LogP,
			LogDefaultScale: profile.LogDefaultScale,
		})
		if err != nil {
			panic(err)
		}

		profile.Name = name
		profile.params = params
		profiles.Profiles[name] = profile
	}
}

func GetProfile(name string) (Profile, error) {
	profile, ok := profiles.Profiles[name]

	if !ok {
		return Profile{}, fmt.Errorf("ckks profile \"%s\" not found", name)
	}

	return profile, nil
}
