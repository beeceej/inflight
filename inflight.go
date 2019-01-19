package inflight

import (
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/cenkalti/backoff"
	uuid "github.com/satori/go.uuid"
)

type (
	// Bucket is a string value which represents the s3 bucket
	Bucket string

	// KeyPath  is a string value which represents the s3 name space objects will be written to
	KeyPath string
)

// ObjectKeyFunc is a function which generates the string to be used as the object key.
type ObjectKeyFunc func() (string, error)

func defaultObjectKeyFunc() (string, error) {
	var (
		u   uuid.UUID
		err error
	)
	if u, err = uuid.NewV4(); err != nil {
		return "", err
	}
	return u.String(), err
}

// Inflight is a structure which provides an interface to retrieving and writing data to s3,
// it doesn't care about what data you're writing, just provides an easy way to get to it
type Inflight struct {
	s3iface.S3API
	Bucket  Bucket
	KeyPath KeyPath

	// ObjectKeyFunc will be called when Inflight#Write(io.ReadSeeker) is invoked.
	// The data will be given the name that this function generates.
	ObjectKeyFunc ObjectKeyFunc
}

// NewInflight Creates a reference to an Inflight struct
func NewInflight(bucket Bucket, keypath KeyPath, s3 s3iface.S3API) *Inflight {
	return &Inflight{
		Bucket:        bucket,
		KeyPath:       keypath,
		S3API:         s3,
		ObjectKeyFunc: defaultObjectKeyFunc,
	}
}

// Write will take the data given and attempt to put it in S3
// It then will return the S3 URI back to the caller so that the data may be passed between callers
func (i *Inflight) Write(data io.ReadSeeker) (ref *Ref, err error) {
	objID, err := i.ObjectKeyFunc()
	if err != nil {
		return nil, backoff.Permanent(err)
	}

	ref = &Ref{
		Bucket: string(i.Bucket),
		Path:   string(i.KeyPath),
		Object: objID,
	}

	err = backoff.Retry(
		i.tryWriteToS3(data, ref.Object),
		backoff.NewExponentialBackOff(),
	)

	if err != nil {
		return nil, err
	}

	return ref, nil
}

func (i *Inflight) tryWriteToS3(data io.ReadSeeker, object string) func() error {
	bucket := string(i.Bucket)
	keyPath := string(i.KeyPath)
	return func() error {
		req := i.PutObjectRequest(&s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(filepath.Join(keyPath, object)),
			Body:        data,
			ContentType: aws.String("application/json"),
		})

		_, err := req.Send()

		if err != nil && aws.IsErrorRetryable(err) {
			return err
		} else if err != nil {
			return backoff.Permanent(err)
		}

		return nil
	}
}

// Get will retrieve the Object at the Bucket and KeyPath from S3.Get
// For instance, if you need the object at `cool-bucket/a/cool/key-path/the-object.json`
// you would say inflight::Get("the-object.json")
func (i *Inflight) Get(object string) ([]byte, error) {
	b := &[]byte{}

	err := backoff.Retry(
		i.tryReadFromS3(object, b),
		backoff.NewExponentialBackOff(),
	)

	if err != nil {
		// returning []byte{} since user may attempt to marshall bytes
		return []byte{}, err
	}

	return *b, err
}

func (i *Inflight) tryReadFromS3(object string, data *[]byte) func() error {
	bucket := string(i.Bucket)
	keyPath := string(i.KeyPath)

	return func() error {
		req := i.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(filepath.Join(keyPath, object)),
		})

		res, err := req.Send()

		if err != nil && aws.IsErrorRetryable(err) {
			return err
		} else if err != nil {
			return backoff.Permanent(err)
		}

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return backoff.Permanent(err)
		}
		defer res.Body.Close()

		*data = append(*data, b...)
		return nil
	}
}
