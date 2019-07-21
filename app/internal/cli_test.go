package internal

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "zero",
			args:    args{[]string{}},
			want:    nil,
			wantErr: true,
		}, {
			name:    "one valid url",
			args:    args{[]string{"http://site.com/lvl1/lvl2/page.html"}},
			want:    []string{"http://site.com/lvl1/lvl2/page.html"},
			wantErr: false,
		}, {
			name: "two valid url",
			args: args{[]string{
				"http://site.com/lvl1/lvl2/page.html",
				"https://site.com/",
			}},
			want: []string{
				"http://site.com/lvl1/lvl2/page.html",
				"https://site.com/",
			},
			wantErr: false,
		}, {
			name:    "one invalid url",
			args:    args{[]string{"site@site.meh/lvl1/lvl2"}},
			want:    nil,
			wantErr: true,
		}, {
			name: "few valid and one invalid url",
			args: args{[]string{
				"http://site.com/lvl1/lvl2/page.html",
				"site@site.meh/lvl1/lvl2",
				"https://site.com/",
			}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgs(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
