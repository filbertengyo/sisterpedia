package bridge

import (
	"math"
	"testing"
)

func TestBridgeRoundTrip(t *testing.T) {
	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := b.Profiles()
	if err != nil || len(profiles) < 2 {
		t.Fatalf("profiles: %v %v", profiles, err)
	}
	kb, err := b.GenKeys("p1")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := b.ValidateKeys(kb.Params, kb.SecretKey, kb.PublicKey, kb.RelinKey); !ok || err != nil {
		t.Fatalf("validate: %v %v", ok, err)
	}
	info, err := b.ParamsInfo(kb.Params)
	if err != nil || info.Profile != "p1" {
		t.Fatalf("params info: %+v %v", info, err)
	}
	ct, err := b.EncryptVector(kb.Params, kb.PublicKey, []float64{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	v, err := b.DecryptVector(kb.Params, kb.SecretKey, ct, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []float64{1, 2, 3} {
		if math.Abs(v[i]-want) > 1e-4 {
			t.Errorf("slot %d = %v", i, v[i])
		}
	}
	ci, err := b.CiphertextInfo(kb.Params, ct)
	if err != nil || ci.Level != profiles[0].MaxLevel || math.Abs(ci.Log2Scale-40) > 1e-9 || ci.Bytes == 0 {
		t.Fatalf("ciphertext info: %+v %v", ci, err)
	}
}

func TestBridgeErrors(t *testing.T) {
	b, _ := New()
	k1, _ := b.GenKeys("p1")
	k2, _ := b.GenKeys("p1")
	cases := map[string]struct {
		f    func() (any, error)
		code string
	}{
		"unknown profile": {func() (any, error) { return b.GenKeys("p9") }, CodeUnknownProfile},
		"bad base64":      {func() (any, error) { return b.EncryptScalar(k1.Params, "@@", 1) }, CodeInvalidPayload},
		"bad params":      {func() (any, error) { return b.EncryptScalar("e30=", k1.PublicKey, 1) }, CodeParamsMismatch},
		"out of range":    {func() (any, error) { return b.EncryptScalar(k1.Params, k1.PublicKey, 1e9) }, CodeValueOutOfRange},
		"keys mismatch":   {func() (any, error) { return b.ValidateKeys(k1.Params, k1.SecretKey, k2.PublicKey, "") }, CodeKeysMismatch},
		"sk as pk":        {func() (any, error) { return b.EncryptScalar(k1.Params, k1.SecretKey, 1) }, CodeInvalidPayload},
	}
	for name, c := range cases {
		r := Wrap(c.f)
		if r.OK || r.Error == nil || r.Error.Code != c.code {
			t.Errorf("%s: got %+v, want code %s", name, r.Error, c.code)
		}
	}
	r := Wrap(func() (any, error) { panic("boom") })
	if r.OK || r.Error.Code != CodeInternal {
		t.Errorf("panic not recovered: %+v", r)
	}
}
