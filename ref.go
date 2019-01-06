package inflight

// Ref represents the path to an object in S3 broken down by bucket, Path, and Object
// Bucket = "my-s3-bucket"
// Path = "some/path/within"
// Object = "an-object-in-s3.json"
// s3://my-s3-bucket/some/path/within/an-object-in-s3.json
type Ref struct {
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
	Object string `json:"object"`
}
