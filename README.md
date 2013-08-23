# bundle_cache

Cache bundle to speed up CI builds for Ruby applications. Written in Go.

## Overview

When running a CI build, you have a clean system. That means that you have to
install dependencies with bundler. It takes a lot of time and its very slow.
Especially when you test suite runs for 20 seconds but bundle install runs for 
more than 2 minutes. Every single time.

`bundle_chache` is here to help. It uploads a bundle tarball to Amazon S3, so 
next time installation will be faster and consume less traffic. Double kill.

## Build

Make sure you have Go 1.1.1 installed. Then run:

```
go get
go build bundle_cache.go
```

## Usage

Amazon S3 account is required. You can export variables for the current session:

```
export S3_ACCESS_KEY=key
export S3_SECRET_KEY=secret
export S3_BUCKET=mybucket
```

Usage:

```
bundler_cache [download|down|upload|up]
```

Or you can invoke command with one time vars:

```
S3_ACCESS_KEY=key \
S3_SECRET_KEY=secret \ 
S3_BUCKET=bucket \
bundle_cache [download|upload]
```
