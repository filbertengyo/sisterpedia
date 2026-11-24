package ckks

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"

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
}

func (p Profile) asLattigoParams() ckks.ParametersLiteral {
	return ckks.ParametersLiteral{
		LogN:            p.LogN,
		LogQ:            p.LogQ,
		LogP:            p.LogP,
		LogDefaultScale: p.LogDefaultScale,
	}
}

func (p Profile) GetParameters() (Parameters, error) {
	params, err := ckks.NewParametersFromLiteral(p.asLattigoParams())
	if err != nil {
		return Parameters{}, err
	}

	return Parameters{value: params}, nil
}

type profilesConfig struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

type Parameters struct {
	value ckks.Parameters
}

func (p Parameters) MarshalBinary() ([]byte, error) {
	return p.value.MarshalBinary()
}

func (p *Parameters) UnmarshalBinary(data []byte) error {
	return p.value.UnmarshalBinary(data)
}

func (p Parameters) MarshalJSON() ([]byte, error) {
	bin, err := p.MarshalBinary()
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(bin)

	return json.Marshal(b64)
}

func (p *Parameters) UnmarshalJSON(data []byte) error {
	var b64 string
	err := json.Unmarshal(data, &b64)
	if err != nil {
		return err
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}

	return p.UnmarshalBinary(bin)
}

var (
	profiles  profilesConfig
	paramsMap map[[32]byte]Profile
)

func init() {
	err := yaml.Unmarshal(config.Profiles, &profiles)
	if err != nil {
		panic(err)
	}

	paramsMap = make(map[[32]byte]Profile)

	for name, profile := range profiles.Profiles {
		profile.Name = name
		profiles.Profiles[name] = profile

		params, err := profile.GetParameters()
		if err != nil {
			panic(err)
		}

		paramsBin, err := params.MarshalBinary()
		if err != nil {
			panic(err)
		}

		digest := sha256.Sum256(paramsBin)
		paramsMap[digest] = profile
	}
}

func GetProfileByName(name string) (Profile, error) {
	profile, ok := profiles.Profiles[name]

	if !ok {
		return Profile{}, errors.New("ckks profile not found")
	}

	return profile, nil
}

func GetProfileByParameters(parameters Parameters) (Profile, error) {
	paramsBin, err := parameters.MarshalBinary()
	if err != nil {
		panic(err)
	}

	digest := sha256.Sum256(paramsBin)
	profile, ok := paramsMap[digest]

	if !ok {
		return Profile{}, errors.New("ckks profile not found")
	}

	return profile, nil
}
