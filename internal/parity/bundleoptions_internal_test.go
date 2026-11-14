package parity

import "testing"

// TestOptionTablesAgree pins that every translator option a bundle profile can
// express also has a CQF CLI flag, so a fresh-CQF comparison never compiles
// with fewer options than the echo-elm side translates with.
func TestOptionTablesAgree(t *testing.T) {
	for opt := range optionProfileFields {
		if optionCLIFlags[opt] == "" {
			t.Errorf("%s has a profile field but no CQF CLI flag", opt)
		}
	}
	for opt := range optionCLIFlags {
		if optionProfileFields[opt] == "" {
			t.Errorf("%s has a CQF CLI flag but no profile field", opt)
		}
	}
}
