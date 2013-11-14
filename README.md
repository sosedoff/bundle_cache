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

### Using on Travis CI

First, encrypt S3 credentials:

```
travis encrypt --add env.global S3_ACCESS_KEY=MYKEY
travis encrypt --add env.global S3_SECRET_KEY=MYSECRET
travis encrypt --add env.global S3_BUCKET=MYBUCKET
```

Then modify your config:

```yml
before_install:
  # replace this with your url, or leave as is
  - wget https://s3.amazonaws.com/bundle-cache-builds/bundle_cache
  - chmod +x ./bundle_cache
  - sudo mv ./bundle_cache /usr/bin
  - bundle_cache download ; echo 1

install:
  - bundle install --deployment --path .bundle

after_script:
  - bundle_cache upload
```

## License

The MIT License (MIT)

Copyright (c) 2013 Dan Sosedoff <dan.sosedoff@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.