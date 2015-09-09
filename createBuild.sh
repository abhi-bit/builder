#!/bin/bash

port=$1
OS=$2
buildXML=$3
buildID=$4

createbuild()
{
    DIR=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
    # list of commands to create build
    ssh -p $port couchbase@localhost "mkdir -p ~/$DIR/builder"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder; repo init -u git://github.com/couchbase/manifest -g all -m $buildXML"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder; repo sync --jobs=20"
    ssh -p $port couchbase@localhost "cd ~/$DIR/builder; cbbuild/scripts/jenkins/couchbase_server/server-linux-build.sh $OS toy-10$buildID.0.0 enterprise 1"

    # upload files to S3
    ssh -p $port couchbase@localhost "cd ~/$DIR/rpmbuild/RPMS/x86_64/; for file in $(ls .); do curl --upload-file $file http://s3.amazonaws.com/customers.couchbase.com/couchbase/; done"
    ssh -p $port couchbase@localhost "cd ~/$DIR/ ; rm -rf builder; rm -rf rpmbuild"
}

createbuild
