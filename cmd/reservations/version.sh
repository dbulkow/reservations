#!/bin/bash

versionfile="version.go"
buildtime=$(date +'%Y-%m-%d %I:%M:%S%p %Z')
githash="github.com/dbulkow/reservations/commit/$(git rev-parse HEAD)"

cat - > ${versionfile} <<EOF
package main

const (
        GitHash = "${githash}"
        BuildTime = "${buildtime}"
)
EOF
