#!/bin/bash

cd $HOME/rpmbuild/RPMS/x86_64/

for i in $(ls .)
do
    curl --upload-file $i http://s3.amazonaws.com/customers.couchbase.com/couchbase/
done
