# bundle_cache

Cache bundle to speed up CI builds for Ruby applications. Written in Go.

## Overview

When running a CI build, you have a clean system. That means that you have to
install dependencies with bundler. It takes a lot of time and its very slow.
Especially when you test suite runs for 20 seconds but bundle install runs for 
more than 2 minutes. Every single time.

`bundle_chache` is here to help. It uploads a bundle tarball to Amazon S3, so 
next time installation will be faster and consume less traffic. Double kill.

Example: 204.17 seconds bundle install reduced to 15.96 seconds

## Build

Make sure you have Go 1.1.1 installed and `GOPATH` is set. Then run:

```
go get
go build bundle_cache.go
```

## Usage

```
Usage:
  bundle_cache [OPTIONS]

Help Options:
  -h, --help=       Show this help message

Application Options:
      --prefix=     Custom archive filename (default: current dir)
      --path=       Path to directory with .bundle (default: current)
      --access-key= S3 Access key
      --secret-key= S3 Secret key
      --bucket=     S3 Bucket name
```

Or you can set S3 credentials for current session:

```
export S3_ACCESS_KEY=MYKEY
export S3_SECRET_KEY=MYSECRET
export S3_BUCKET=MYBUCKET
```

And then run (within project directory):

```
bundle_cache download
bundle_cache upload
```

## License

The MIT License (MIT)

Copyright (c) 2013-2014 Dan Sosedoff <dan.sosedoff@gmail.com>