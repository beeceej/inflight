# Inflight
[![Build Status](https://travis-ci.org/beeceej/inflight.svg?branch=master)](https://travis-ci.org/beeceej/inflight)
[![codecov](https://codecov.io/gh/beeceej/inflight/branch/master/graph/badge.svg)](https://codecov.io/gh/beeceej/inflight)

Inflight is a simple package which abstracts away writing to and from S3. It handles retries (with an exponential back off algorithm), and can be configured to write to any S3 bucket.

An Example of where this package could be useful is when the data you're working with is too large to be manually passed through. This provides a simple way to ship references to data between functions in a state machine.

Example usage.

## Lambda 1:
```go
package main

var infl *inflight.Infight

func init() {
  bucket := os.Getenv("INFLIGHT_BUCKET")
  path := os.Getenv("INFLIGHT_PATH")
  s3 := //an s3 client
  infl = inflight.NewInflight(Bucket(bucket), KeyPath(path), s3)
}

func handler(event interface{}) *inflight.Ref {
  dataWhichIsOverTheAwsLimit := bytes.NewReader(...)
  return infl.Write(dataWhichIsOverTheAwsLimit)
}

func main() {
  lambda.Start(handler)
}
```

## Lambda 2:
```go
package main

var infl *inflight.Infight

func init() {
  bucket := os.Getenv("INFLIGHT_BUCKET")
  path := os.Getenv("INFLIGHT_PATH")
  s3 := //an s3 client
  infl = inflight.NewInflight(Bucket(bucket), KeyPath(path), s3)
}

func handler(event *inflight.Ref) *inflight.Ref {
  b, err := infl.Get(event.Object)
  // ...
  // Do Some stuff with your data
  // ...

  dataWhichIsOverTheAwsLimit := doSomeStuffWithYourData(b)

  // Write the data back
  return infl.Write(dataWhichIsOverTheAwsLimit)
}

func main() {
  lambda.Start(handler)
}
```



