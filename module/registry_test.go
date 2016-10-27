package module

import "testing"

func TestRegister(t *testing.T) {
	name := "UltraMegaSuperCrusher"
	expected := &mockPatcher{}

	Register(name, func(c Config) (Patcher, error) {
		expected.value = c["key"].(string)
		return expected, nil
	})

	if init, err := Lookup(name); err == nil {
		p, err := init(Config{"key": "hello"})
		if err != nil {
			t.Error(err)
		}
		if expected != p {
			t.Errorf("expected=%v actual=%v", expected, p)
		}
	} else {
		t.Error("lookup should have been successful for %s", name)
	}

	if _, err := Lookup("unknown"); err == nil {
		t.Error("lookup should have been failed for unknown module")
	}
}

type mockPatcher struct {
	value string
}

func (p mockPatcher) Patch(string, interface{}) error { return nil }
func (p mockPatcher) Output(string) (Reader, error)   { return nil, nil }
func (p mockPatcher) Reset() error                    { return nil }
