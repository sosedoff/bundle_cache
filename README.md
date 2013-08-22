# bundle-cache

Sync bundler files with Amazon S3

## Build

Make sure you have Go 1.1.1 installed. Then run:

```
go get
go build bundle_cache.go
```

## Usage

```
bundler-cache [download|down|upload|up]
```

Make sure to provide S3 credentials:

```
S3_ACCESS_KEY=foo S3_SECRET_KEY=bar S3_BUCKET=bundle_cache bundler-cache [action]
```