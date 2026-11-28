package ckks

import (
	"encoding/json"
	"testing"
)

func TestLoadP1(t *testing.T) {
	_, err := GetProfile("p1")
	if err != nil {
		t.Errorf("GetProfile(\"p1\") returned an error: %v", err)
	}
}

func TestLoadP2(t *testing.T) {
	_, err := GetProfile("p2")
	if err != nil {
		t.Errorf("GetProfile(\"p2\") returned an error: %v", err)
	}
}

func TestLoadUnknownProfile(t *testing.T) {
	_, err := GetProfile("p3")
	if err == nil {
		t.Errorf("GetProfile(\"p3\") didn't return an error")
	}
}

func TestP1Security(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("GetProfile(\"p1\") returned an error: %v", err)
		return
	}

	params := profile.params

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
	profile, err := GetProfile("p2")
	if err != nil {
		t.Errorf("GetProfile(\"p2\") returned an error: %v", err)
		return
	}

	params := profile.params

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

func TestProfileBinaryMarshal(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	payload, err := profile.MarshalBinary()
	if err != nil {
		t.Errorf("error while encoding profile: %v", err)
		return
	}

	var decodedProfile Profile
	err = decodedProfile.UnmarshalBinary(payload)
	if err != nil {
		t.Errorf("error while decoding profile: %v", err)
		return
	}

	if profile.Name() != decodedProfile.Name() {
		t.Error("decoded profile does not match original")
		return
	}
}

func TestProfileJsonMarshal(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	payload, err := json.Marshal(profile)
	if err != nil {
		t.Errorf("error while encoding profile: %v", err)
		return
	}

	var decodedProfile Profile
	err = json.Unmarshal(payload, &decodedProfile)
	if err != nil {
		t.Errorf("error while decoding profile: %v", err)
		return
	}

	if profile.Name() != decodedProfile.Name() {
		t.Error("decoded profile does not match original")
		return
	}
}
