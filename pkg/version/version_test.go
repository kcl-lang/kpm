// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.
package version

import "testing"

// Deprecated: This function will be removed in a future version.
func TestGetVersionInStr(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "test get version in string",
			want: "0.11.0-alpha.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVersionInStr(); got != tt.want {
				t.Errorf("GetVersionInStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
