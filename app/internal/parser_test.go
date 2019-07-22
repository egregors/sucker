package internal

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHtmlParser_isValidExt(t *testing.T) {
	type fields struct {
		pages chan string
		links chan string
		exts  []string
	}
	type args struct {
		ext string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"valid ext",
			fields{nil, nil, []string{"webm", "mp4"}},
			args{"/a/b/c/d.mp4"},
			true,
		}, {
			"invalid ext",
			fields{nil, nil, []string{"webm", "mp4"}},
			args{"/a/b/c/d.mkv"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HtmlParser{
				pages: tt.fields.pages,
				links: tt.fields.links,
				exts:  tt.fields.exts,
			}
			if got := p.isValidHref(tt.args.ext); got != tt.want {
				t.Errorf("HtmlParser.isValidHref() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewHtmlParser(t *testing.T) {
	type args struct {
		pagesUrl []string
		exts     []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"empty exts",
			args{nil, []string{}},
			[]string{"webm", "mp4"},
		}, {
			"custom exts",
			args{nil, []string{"png", "img"}},
			[]string{"png", "img"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHtmlParser(tt.args.pagesUrl, tt.args.exts).exts; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHtmlParser().exts = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHtmlParser_GetLinks(t *testing.T) {
	tsEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, _ := filepath.Abs("parser_test_fixtures/empty.html")
		data, _ := ioutil.ReadFile(path)
		_, _ = fmt.Fprintln(w, string(data))
	}))
	defer tsEmpty.Close()

	// love this name ;)
	ts2webm1mp4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, _ := filepath.Abs("parser_test_fixtures/ts2webm1mp4.html")
		data, _ := ioutil.ReadFile(path)
		_, _ = fmt.Fprintln(w, string(data))
	}))
	defer ts2webm1mp4.Close()

	type fields struct {
		pagesUrls []string
		exts      []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string // urls from channel
	}{
		{
			"page without hrefs",
			fields{[]string{tsEmpty.URL}, nil},
			nil,
		}, {
			"page without hrefs",
			fields{[]string{ts2webm1mp4.URL}, nil},
			[]string{
				fmt.Sprintf("%s/a/b/c/d.webm", ts2webm1mp4.URL),
				fmt.Sprintf("%s/a/b/c/d2.webm", ts2webm1mp4.URL),
				fmt.Sprintf("%s/a/b/c/d3.mp4", ts2webm1mp4.URL),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHtmlParser(tt.fields.pagesUrls, tt.fields.exts)
			if got := p.GetLinks(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HtmlParser.GetLinks() = %v, want %v", got, tt.want)
			}
		})
	}
}
