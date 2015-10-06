#!/bin/bash

packageDir=$1
OS=$(lsb_release -d | awk '{print $2}')

if [ "$OS" == "CentOS" ]
then
        cd $HOME/rpmbuild/RPMS/x86_64/
        ext="*.rpm"
else
        cd $packageDir/builder/
        ext="*.deb"
fi

for file in $(ls $ext)
do
    echo "Uploading to S3:" $file
    curl --upload-file $file http://s3.amazonaws.com/customers.couchbase.com/couchbase/
done
