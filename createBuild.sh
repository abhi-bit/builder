#!/bin/bash

port=$1
OS=$2
buildXML=$3

createbuild()
{
    # list of commands to create build
    ssh -p $port couchbase@localhost "mkdir builder"
    ssh -p $port couchbase@localhost "cd builder"
    ssh -p $port couchbase@localhost "cd builder; repo init -u git://github.com/couchbase/manifest -g all -m $buildXML"
    ssh -p $port couchbase@localhost "cd builder; repo sync --jobs=20"
    ssh -p $port couchbase@localhost "cd builder; cbbuild/scripts/jenkins/couchbase_server/server-linux-build.sh $OS 1010.0.0 enterprise 1"
    ssh -p $port couchbase@localhost "bash /home/couchbase/upload.sh"
    ssh -p $port couchbase@localhost "cd /home/couchbase/; rm -rf builder"
}

createbuild