package inflight

import (
	"bytes"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
)

type mocks3Basic struct {
	s3iface.S3API
}

type mocks3GetObjectRequestError struct {
	s3iface.S3API
}

type mocks3GetObjectRequestReturnBytes struct {
	s3iface.S3API
	bytesToReturn []byte
}

type mocks3GetObjectRequestRetryableErrorReturnBytesAfterSecondAttempt struct {
	s3iface.S3API
	times         int
	bytesToReturn []byte
}

type mocks3PutObjectRequestRetryableErrorExpectSuccessAfterSecondAttempt struct {
	s3iface.S3API
	count      int
	givenBytes []byte
}

func (m *mocks3PutObjectRequestRetryableErrorExpectSuccessAfterSecondAttempt) PutObjectRequest(input *s3.PutObjectInput) s3.PutObjectRequest {
	if m.count == 0 {
		m.count++
		return s3.PutObjectRequest{
			Request: &aws.Request{Data: &s3.PutObjectOutput{}, Error: awserr.New("RequestTimeout", "", errors.New(""))}}
	}

	return s3.PutObjectRequest{
		Request: &aws.Request{
			Data:  &s3.PutObjectOutput{},
			Error: nil,
		},
	}
}

func TestNewInflightGivenBucketAndKeyExpectCorrectValues(t *testing.T) {
	s3 := new(mocks3Basic)
	givenBucket := Bucket("bucket")
	givenKeyPath := KeyPath("key/path")

	expected := &Inflight{
		Bucket:        Bucket("bucket"),
		KeyPath:       KeyPath("key/path"),
		S3API:         s3,
		ObjectKeyFunc: ObjectKeyFunc(defaultObjectKeyFunc),
	}

	actual := NewInflight(givenBucket, givenKeyPath, s3)

	if cmp.Equal(expected, actual) {
		t.Fail()
	}
}

func (m mocks3GetObjectRequestError) GetObjectRequest(*s3.GetObjectInput) s3.GetObjectRequest {
	return s3.GetObjectRequest{
		Request: &aws.Request{Data: &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewReader([]byte{})),
		}, Error: errors.New("")},
	}
}

func TestGetGivenObjectNotExistExpectError(t *testing.T) {
	inflight := NewInflight(Bucket(""), KeyPath(""), new(mocks3GetObjectRequestError))
	_, err := inflight.Get("object")
	if err == nil {
		t.Fail()
	}
}

func (m mocks3GetObjectRequestReturnBytes) GetObjectRequest(*s3.GetObjectInput) s3.GetObjectRequest {
	return s3.GetObjectRequest{
		Request: &aws.Request{Data: &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewReader(m.bytesToReturn))}, Error: nil},
	}
}

func TestGetGivenObjectExistExpectCorrectBytes(t *testing.T) {
	expectedBytes := []byte("Hello, World!")
	inflight := NewInflight(Bucket(""), KeyPath(""), &mocks3GetObjectRequestReturnBytes{
		bytesToReturn: expectedBytes,
	})

	anyString := ""
	actualBytes, err := inflight.Get(anyString)

	if err != nil {
		t.Fail()
	}
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Fail()
	}
}

func (m *mocks3GetObjectRequestRetryableErrorReturnBytesAfterSecondAttempt) GetObjectRequest(*s3.GetObjectInput) s3.GetObjectRequest {
	if m.times == 0 {
		m.times++
		return s3.GetObjectRequest{
			Request: &aws.Request{Data: &s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte{}))}, Error: awserr.New("RequestTimeout", "", errors.New(""))},
		}
	}

	return s3.GetObjectRequest{
		Request: &aws.Request{Data: &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewReader(m.bytesToReturn))},
			Error: nil},
	}
}
func TestGetGivenObjectExistExpectRetryableErrorThenBytesReturned(t *testing.T) {
	expectedBytes := []byte("Hello, World!")
	inflight := NewInflight(Bucket(""), KeyPath(""), &mocks3GetObjectRequestRetryableErrorReturnBytesAfterSecondAttempt{
		bytesToReturn: expectedBytes,
	})

	anyString := ""
	actualBytes, err := inflight.Get(anyString)

	if err != nil {
		t.Fail()
	}
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Fail()
	}
}

func TestWriteGivenSomeBytesExpectRetryableErrorThenIdentifierReturned(t *testing.T) {
	givenBytes := []byte("hi")
	givenBucket := Bucket("a_bucket")

	givenKeyPath := KeyPath("a/key/path")
	s3 := &mocks3PutObjectRequestRetryableErrorExpectSuccessAfterSecondAttempt{
		givenBytes: givenBytes,
	}

	inflight := NewInflight(givenBucket, givenKeyPath, s3)
	actualRef, err := inflight.Write(givenBytes)
	if err != nil {
		t.Fail()
	}

	if actualRef.Bucket != string(givenBucket) {
		t.Fail()
	}

	if actualRef.Path != string(givenKeyPath) {
		t.Fail()
	}

	if actualRef.Object == "" {
		t.Fail()
	}
}

func TestWriteGivenSomeBytesButUUIDReturnsErrorExpectPermanentError(t *testing.T) {
	givenBytes := []byte("hi")
	givenBucket := Bucket("a_bucket")

	givenKeyPath := KeyPath("a/key/path")
	s3 := &mocks3PutObjectRequestRetryableErrorExpectSuccessAfterSecondAttempt{
		givenBytes: givenBytes,
	}

	inflight := NewInflight(givenBucket, givenKeyPath, s3)
	inflight.ObjectKeyFunc = func(b []byte) (string, error) {
		return "", errors.New("")
	}

	_, err := inflight.Write(givenBytes)
	if err == nil {
		t.Fail()
	}
}

func TestWriteGivenSomeBytesButUUIDReturnsErrorExpectStringFromGenerator(t *testing.T) {
	givenBytes := []byte("hi")
	givenBucket := Bucket("a_bucket")

	givenKeyPath := KeyPath("a/key/path")
	s3 := &mocks3PutObjectRequestRetryableErrorExpectSuccessAfterSecondAttempt{
		givenBytes: givenBytes,
	}

	inflight := NewInflight(givenBucket, givenKeyPath, s3)
	inflight.ObjectKeyFunc = func(b []byte) (string, error) {
		return "from_the_func", nil
	}

	ref, err := inflight.Write(givenBytes)
	if err != nil && ref.Object != "from_the_func" {
		t.Fail()
	}
}
