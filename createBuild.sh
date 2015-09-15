#!/bin/bash

port=$1
OS=$2
repo=$3 # eg, git://github.com/couchbase/manifest
buildXML=$4
buildID=$5
DIR=$6

createbuild()
{
    # list of commands to create build
    ssh -p $port couchbase@localhost "mkdir -p ~/$DIR/builder"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder; repo init -u $repo -g all -m $buildXML"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder; repo sync --jobs=20"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder; cbbuild/scripts/jenkins/couchbase_server/server-linux-build.sh $OS toy-10$buildID.0.0 enterprise 1"

    # upload files to S3
    ssh -p $port couchbase@localhost "bash ~/upload.sh"
    ssh -p $port couchbase@localhost "cd; rm -rf ~/$DIR; rm -rf rpmbuild"
}

createbuild
