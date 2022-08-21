package kingpin

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestGetClosestMatch(t *testing.T) {
	options := []string{
		"ssh", "ls", "all", "v", "debug", "scp",
	}

	type args struct {
		arg      string
		options  []string
		maxEdits int
	}
	tests := []struct {
		args args
		want string
	}{
		{
			args: args{
				arg:      "sss",
				options:  options,
				maxEdits: 3,
			},
			want: "ssh",
		},
		{
			args: args{
				arg:      "ssh",
				options:  options,
				maxEdits: 3,
			},
			want: "ssh",
		},
		{
			args: args{
				arg:      "ll",
				options:  options,
				maxEdits: 3,
			},
			want: "ls",
		},
		{
			args: args{
				arg:      "sl",
				options:  options,
				maxEdits: 3,
			},
			want: "ls",
		},
		{
			args: args{
				arg:      "deugk",
				options:  options,
				maxEdits: 3,
			},
			want: "debug",
		},
		{
			args: args{
				arg:      "al",
				options:  options,
				maxEdits: 3,
			},
			want: "all",
		},
		{
			args: args{
				arg:      "vvvvv",
				options:  options,
				maxEdits: 3, // exceeds max edits
			},
			want: "",
		},
		{
			args: args{
				arg:      "vvvv",
				options:  options,
				maxEdits: 3,
			},
			want: "v",
		},
		{
			args: args{
				arg:      "cp",
				options:  options,
				maxEdits: 3,
			},
			want: "scp",
		},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("input %s expected %s", tt.args.arg, tt.want)
		t.Run(name, func(t *testing.T) {
			// shuffle the available options. We don't want to be order dependent.
			rand.Shuffle(len(tt.args.options), func(i, j int) {
				tt.args.options[i], tt.args.options[j] = tt.args.options[j], tt.args.options[i]
			})
			if got := GetClosestMatch(tt.args.arg, tt.args.options, tt.args.maxEdits); got != tt.want {
				t.Errorf("GetClosestMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
