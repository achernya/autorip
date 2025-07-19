package imdb

import (
	"fmt"
	"net"
	"net/http"
	"testing"
)

func TestFetchSucceeds(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close() //nolint:errcheck

	addr := listener.Addr().(*net.TCPAddr)
	datasetSource = fmt.Sprintf("http://localhost:%d/", addr.Port)

	src := newTmpDir(t)
	defer src.Cleanup()
	if err := copyTestData(src.dir); err != nil {
		t.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.Dir(src.dir)))
	go func() {
		select {
		case <-t.Context().Done():
		default:
			http.Serve(listener, nil) //nolint:errcheck
		}
	}()

	dst := newTmpDir(t)
	defer dst.Cleanup()
	if err := Fetch(t.Context(), dst.dir); err != nil {
		t.Fatal(err)
	}
}
