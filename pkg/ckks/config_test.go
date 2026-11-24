package ckks

import (
	"testing"

	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

func TestLoadP1(t *testing.T) {
	_, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("GetProfile(\"p1\") returned an error: %v", err)
	}
}

func TestLoadP2(t *testing.T) {
	_, err := GetProfileByName("p2")
	if err != nil {
		t.Errorf("GetProfile(\"p2\") returned an error: %v", err)
	}
}

func TestLoadUnknownProfile(t *testing.T) {
	_, err := GetProfileByName("p3")
	if err == nil {
		t.Errorf("GetProfile(\"p3\") didn't return an error")
	}
}

func TestLoadProfileFromParam(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("GetProfile returned an error: %v", err)
		return
	}

	params, err := profile.GetParameters()
	if err != nil {
		t.Errorf("GetParameters returned an error: %v", err)
		return
	}

	retrievedProfile, err := GetProfileByParameters(params)
	if err != nil {
		t.Errorf("GetProfileByParameters returned an error: %v", err)
		return
	}

	if profile.Name != retrievedProfile.Name {
		t.Error("retrieved profile does not match original profile")
	}
}

func TestP1Security(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("GetProfile(\"p1\") returned an error: %v", err)
		return
	}

	params, err := ckks.NewParametersFromLiteral(profile.asLattigoParams())
	if err != nil {
		t.Errorf("Getting params returned an error: %v", err)
		return
	}

	if params.MaxLevel() < 2 {
		t.Errorf("Profile p1's max level is less than 2")
		return
	}

	if params.LogN() != 13 {
		t.Errorf("Profile p1's LogN is %d, expected 13", params.LogN())
		return
	}

	if params.LogQP() > 218 {
		t.Errorf("Profile p1's LogQP value exceeds 218 (%f)", params.LogQP())
		return
	}
}

func TestP2Security(t *testing.T) {
	profile, err := GetProfileByName("p2")
	if err != nil {
		t.Errorf("GetProfile(\"p2\") returned an error: %v", err)
		return
	}

	params, err := ckks.NewParametersFromLiteral(profile.asLattigoParams())
	if err != nil {
		t.Errorf("Getting params returned an error: %v", err)
		return
	}

	if params.MaxLevel() < 2 {
		t.Errorf("Profile p2's max level is less than 2")
		return
	}

	if params.LogN() != 14 {
		t.Errorf("Profile p2's LogN is %d, expected 14", params.LogN())
		return
	}

	if params.LogQP() > 438 {
		t.Errorf("Profile p2's LogQP value exceeds 438 (%f)", params.LogQP())
		return
	}
}
