//go:generate protoc --go_out=. --go_opt=paths=source_relative --proto_path=. --go_opt=default_api_level=API_OPAQUE imdb.proto
package proto
