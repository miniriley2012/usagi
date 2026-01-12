//go:build generate

//go:generate sh -c "go run ../../../tools/generate_range_table.go ID_Start.txt ID_Continue.txt > ucd.go"
package generate
