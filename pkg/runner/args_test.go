package runner

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Flag
		wantErr bool
	}{
		{
			name: "Test with ExternalPkg",
			args: []string{"-E", "pkg1=pkg1_path", "-E", "pkg2=pkg2_path"},
			want: Flag{
				ExternalPkg: []string{"pkg1=pkg1_path", "pkg2=pkg2_path"},
				Options:     nil,
				Overrides:   nil,
				DisableNone: false,
				SortKeys:    false,
				Settings:    "",
			},
			wantErr: false,
		},
		{
			name: "Test with Options",
			args: []string{"-D", "option1=option1_value", "-D", "option2=option2_value"},
			want: Flag{
				ExternalPkg: nil,
				Options:     []string{"option1=option1_value", "option2=option2_value"},
				Overrides:   nil,
				DisableNone: false,
				SortKeys:    false,
				Settings:    "",
			},
			wantErr: false,
		},
		{
			name: "Test with Overrides",
			args: []string{"-O", "override1=override1_value", "-O", "override2=override2_value"},
			want: Flag{
				ExternalPkg: nil,
				Options:     nil,
				Overrides:   []string{"override1=override1_value", "override2=override2_value"},
				DisableNone: false,
				SortKeys:    false,
				Settings:    "",
			},
			wantErr: false,
		},
		{
			name: "Test with DisableNone",
			args: []string{"-n"},
			want: Flag{
				ExternalPkg: nil,
				Options:     nil,
				Overrides:   nil,
				DisableNone: true,
				SortKeys:    false,
				Settings:    "",
			},
			wantErr: false,
		},
		{
			name: "Test with SortKeys",
			args: []string{"-k"},
			want: Flag{
				ExternalPkg: nil,
				Options:     nil,
				Overrides:   nil,
				DisableNone: false,
				SortKeys:    true,
				Settings:    "",
			},
			wantErr: false,
		},
		{
			name: "Test with Settings",
			args: []string{"-Y", "settings.yaml"},
			want: Flag{
				ExternalPkg: nil,
				Options:     nil,
				Overrides:   nil,
				DisableNone: false,
				SortKeys:    false,
				Settings:    "settings.yaml",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				assert.Equal(t, got.ExternalPkg, tt.want.ExternalPkg)
				assert.Equal(t, got.Options, tt.want.Options)
				assert.Equal(t, got.Overrides, tt.want.Overrides)
				assert.Equal(t, got.DisableNone, tt.want.DisableNone)
				assert.Equal(t, got.SortKeys, tt.want.SortKeys)
				assert.Equal(t, got.Settings, tt.want.Settings)
				t.Errorf("ParseArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
