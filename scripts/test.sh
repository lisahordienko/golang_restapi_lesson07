#!/bin/sh
set -eu

goos=$(go env GOOS)
goversion=$(go env GOVERSION)

case "$goos:$goversion" in
	darwin:go1.22.*)
		exec go test -ldflags=-linkmode=external "$@"
		;;
	*)
		exec go test "$@"
		;;
esac