package module

import "testing"

type mockProvider struct{}

func (p mockProvider) Read(Frame)     {}
func (p mockProvider) Reader() Reader { return p }

func TestExpose(t *testing.T) {
	io := &IO{}
	if err := io.Expose(
		[]*In{{Name: "input"}},
		[]*Out{{Name: "output", Provider: mockProvider{}}},
	); err != nil {
		t.Error(err)
	}

	if _, ok := io.Inputs()["input"]; !ok {
		t.Error("input not registered")
	}
	if _, err := io.Output("output"); err != nil {
		t.Error(err)
	}
}

func TestPatching(t *testing.T) {
	one := &IO{}
	if err := one.Expose(
		[]*In{{Name: "input"}},
		[]*Out{{Name: "output", Provider: mockProvider{}}},
	); err != nil {
		t.Error(err)
	}

	two := &IO{}
	if err := two.Expose(
		[]*In{{Name: "input"}},
		[]*Out{{Name: "output", Provider: mockProvider{}}},
	); err != nil {
		t.Error(err)
	}

	if err := one.Patch("input", Port{two, "output"}); err != nil {
		t.Error(err)
	}

	if err := one.Patch("input", Port{two, "unknown"}); err == nil {
		t.Error("patch to unknown output port did not fail")
	}

	if actual, expected := two.OutputsActive(), 1; actual != expected {
		t.Errorf("expected=%v actual=%v", expected, actual)
	}

	if err := two.Reset(); err != nil {
		t.Error(err)
	}
}

func TestExposeDuplicateIn(t *testing.T) {
	io := &IO{}
	if err := io.Expose(
		[]*In{{Name: "input"}, {Name: "input"}},
		[]*Out{{Name: "output", Provider: mockProvider{}}},
	); err == nil {
		t.Error("duplicate input should have failed")
	}
}

func TestExposeDuplicateOut(t *testing.T) {
	io := &IO{}
	if err := io.Expose(
		[]*In{{Name: "input"}},
		[]*Out{
			{Name: "output", Provider: mockProvider{}},
			{Name: "output", Provider: mockProvider{}},
		},
	); err == nil {
		t.Error("duplicate output should have failed")
	}
}
