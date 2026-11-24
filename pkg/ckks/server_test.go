package ckks

import "testing"

func TestNewServerCrypto(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	_, err = publicKeyset.NewServerCrypto()
	if err != nil {
		t.Errorf("error obtaining server crypto: %v", err)
		return
	}
}

func TestEncryptScalar(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	serverCrypto, err := publicKeyset.NewServerCrypto()
	if err != nil {
		t.Errorf("error obtaining server crypto: %v", err)
		return
	}

	_, err = serverCrypto.EncryptScalar(5761)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}
}

func TestEncryptScalarAt(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	serverCrypto, err := publicKeyset.NewServerCrypto()
	if err != nil {
		t.Errorf("error obtaining server crypto: %v", err)
		return
	}

	_, err = serverCrypto.EncryptScalarAt(5761, 1)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}
}

func TestEncryptVector(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	serverCrypto, err := publicKeyset.NewServerCrypto()
	if err != nil {
		t.Errorf("error obtaining server crypto: %v", err)
		return
	}

	_, err = serverCrypto.EncryptVector([]float64{5, 7, 6, 1})
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}
}

func TestEncryptVectorAt(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	serverCrypto, err := publicKeyset.NewServerCrypto()
	if err != nil {
		t.Errorf("error obtaining server crypto: %v", err)
		return
	}

	_, err = serverCrypto.EncryptVectorAt([]float64{5, 7, 6, 1}, 1)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}
}
