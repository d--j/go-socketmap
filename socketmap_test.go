package socketmap

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func Test_write(t *testing.T) {
	big := make([]byte, MaxSize)
	wantBig := strings.Repeat("\x00", MaxSize)
	tests := []struct {
		name    string
		chunks  [][]byte
		want    string
		wantErr bool
	}{
		{"empty", [][]byte{}, "0:,", false},
		{"single chunk", [][]byte{[]byte("abc")}, "3:abc,", false},
		{"multiple chunks", [][]byte{[]byte("abc"), []byte("def"), []byte("abc")}, "9:abcdefabc,", false},
		{"much data", [][]byte{big}, fmt.Sprintf("%d:%s,", MaxSize, wantBig), false},
		{"too much data", [][]byte{[]byte("abc"), big}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w := net.Pipe()
			wg := sync.WaitGroup{}
			wg.Add(1)
			got := ""
			go func() {
				b, _ := io.ReadAll(r)
				got = string(b)
				wg.Done()
			}()
			if err := write(w, tt.chunks...); (err != nil) != tt.wantErr {
				t.Errorf("write() error = %v, want %v", err, tt.wantErr)
			}
			w.Close()
			wg.Wait()
			if got != tt.want {
				t.Errorf("write() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_read(t *testing.T) {
	tests := []struct {
		name    string
		wire    string
		want    string
		wantErr bool
	}{
		{"empty", "", "", true},
		{"part1", "123", "", true},
		{"part2", "123:", "", true},
		{"part3", "1:1", "", true},
		{"part4", "1:1:", "", true},
		{"part5", "123456789:1:", "", true},
		{"wrong size1", "a:1,", "", true},
		{"wrong size2", "30:1,", "", true},
		{"wrong size3", fmt.Sprintf("%d:%s,", MaxSize+1, strings.Repeat(" ", MaxSize+1)), "", true},
		{"ok empty", "0:,", "", false},
		{"ok", "1:1,", "1", false},
		{"big", fmt.Sprintf("%d:%s,", MaxSize, strings.Repeat(" ", MaxSize)), strings.Repeat(" ", MaxSize), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w := net.Pipe()
			go func() {
				w.Write([]byte(tt.wire))
				w.Close()
			}()
			gotB, err := read(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := string(gotB)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("read() got = %v, want %v", got, tt.want)
			}
		})
	}
}
