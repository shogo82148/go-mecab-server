#!/bin/bash

set -ue

ROOT=$(cd $(dirname $0);pwd)
LATEST_VERSION=`curl -s https://api.github.com/repos/neologd/mecab-ipadic-neologd/git/refs/heads/master | jq .object.sha`
NOW_VERSION=`curl -s http://localhost:9001/?parsers=mecab_neologd | jq .neologd_version`

echo "$LATEST_VERSION"
echo "$NOW_VERSION"

if [[ "$LATEST_VERSION" != "$NOW_VERSION" ]]; then
    cd $ROOT
    docker build -t mecab --no-cache .
    kill -HUP `supervisorctl pid mecab-server`
fi
