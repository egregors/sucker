package downloader

import "testing"

func Test_getFileNameFromURL(t *testing.T) {
	type args struct {
		l string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "valid 1",
			args: args{
				l: "/b/src/236934044/16097616551560.webm ",
			},
			want: "16097616551560.webm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFileNameFromURL(tt.args.l); got != tt.want {
				t.Errorf("getFileNameFromURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
