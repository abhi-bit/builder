#!/bin/bash

cd $HOME/rpmbuild/RPMS/x86_64/

for file in $(ls .)
do
    echo "Uploading to S3:" $file
    curl --upload-file $file http://s3.amazonaws.com/customers.couchbase.com/couchbase/
done
