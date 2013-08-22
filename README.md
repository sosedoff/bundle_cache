# bundle-cache

Cache bundle to speed up CI deployments for Ruby applications. Written in Go.

## Build

Make sure you have Go 1.1.1 installed. Then run:

```
go get
go build bundle_cache.go
```

## Usage

You will need to have Amazon S3 account. Simply export it for the session:

```
export S3_ACCESS_KEY=key
export S3_SECRET_KEY=secret
export S3_BUCKET=mybucket
```

Then run:

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