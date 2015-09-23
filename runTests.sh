#!/bin/bash

# hardcoding port to centos6 instance
port=$1
repo=$2
buildXML=$3
nodeCount=$4
iniFile=$5
confFile=$6
DIR=$7

fireTests()
{
    ssh -p $port couchbase@localhost "mkdir -p ~/$DIR/tests"
    ssh -p $port couchbase@localhost "cd ~/$DIR/tests;repo init -u $repo -g all -m $buildXML"
    ssh -p $port couchbase@localhost "cd ~/$DIR/tests; repo sync --jobs=20"
    ssh -p $port couchbase@localhost "cd ~/$DIR/tests/ns_server; ./cluster_run -n $nodeCount &>/dev/null &; sleep 60; ./cluster_connect -n $nodeCount"
    ssh -p $port couchbase@localhost "cd ~/$DIR/tests/testrunner; ./testrunner -i $iniFile -c $confFile"
    ssh -p $port couchbase@localhost "cd ~/DIR/tests/; ./install/bin/cbcollect_info -v $DIR.zip"

    # upload files to S3
    ssh -p $port couchbase@localhost "curl -v --upload-file $DIR.zip https://s3.amazonaws.com/customers.couchbase.com/couchbase/"
    ssh -p $port couchbase@localhost "rm -rf ~/$DIR "

}

fireTests
