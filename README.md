### Automated toy-builder:


### Steps:

* Have the OS and toy build xml file name handy, you will need this
  to fire request to web-service.
* To get the list of supported platforms for creating toy builds:

```
> curl http://docker:8080/OS
[“centos6”,”centos7”,”ubuntu12”,”ubuntu14”,”debian7”]
```

* To generate toy-build for centos6 using `toy/toy-abhi.xml`,
  API call would look like:

```
> curl http://docker:8080/build/centos6\?xmlfile\=toy/xy.xml
````

* To generate toy-build from a specific manifest repostiory,
  API call would look like:

```
> curl http://docker:8080/build/centos6\?xmlfile\=toy/xy.xml\&repo\=git://github.com/myname/manifest
````

* Typically it takes around 15 mins to generate toy-build for one platform
  from *docker* machine. Once done it will upload the file to S3:

  ```
  http://s3.amazonaws.com/couchbase-latestbuilds/couchbase/
  ```

Right now it’s running on *docker* machine in BLR office.
It’s easy to set this up on any other machine as well, all you need is
to fire up docker instances for OSes you plan to create builds for and
then pass port # details in `app.go`
