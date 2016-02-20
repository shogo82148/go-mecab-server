#!/bin/bash

set -ue

ROOT=$(cd $(dirname $0);pwd)

cd $ROOT
exec start_server --port=9001 --interval=30 --kill-old-delay=60 --pid-file=app.pid -- $ROOT/start-worker.sh
