package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/google/pprof/driver"
	"github.com/minio/minio-go"
	"github.com/sbueringer/pprof-exporter/pkg/pprof"
	"net"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

var defaultUrl = "localhost:9100/debug/pprof"
var defaultProfile = "heap"
var defaultDebug = "1"
var defaultObjectPrefix = "profiles"

var profileDurationSeconds int
var profileDelaySeconds int

var s3Endpoint string
var s3AccessKeyID string
var s3SecretAccessKey string
var s3Region string
var s3Bucket string
var keepCount int

func init() {
	flag.IntVar(&profileDurationSeconds, "profile-duration-seconds", 30, "duration of profiles.")
	flag.IntVar(&profileDelaySeconds, "profile-delay-seconds", 30, "delay between profiles.")

	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "s3 Endpoint.")
	flag.StringVar(&s3AccessKeyID, "s3-access-key-id", os.Getenv("S3_ACCESS_KEY_ID"), "s3 Access Key ID.")
	flag.StringVar(&s3SecretAccessKey, "s3-secret-access-key", os.Getenv("S3_SECRET_ACCESS_KEY"), "s3 Secret Access Key.")
	flag.StringVar(&s3Region, "s3-region", "", "s3 Region.")
	flag.StringVar(&s3Bucket, "s3-bucket", "default", "s3 Bucket.")
	flag.IntVar(&keepCount, "keep-count", 10, "count profiles to keep.")
}

func main() {
	flag.Parse()

	client, err := initMinio()
	if err != nil {
		panic(err)
	}

	////TODO maybe parse possible profiles
	//resp, err := http.Get("http://" + defaultUrl)
	//if err != nil {
	//	panic(err)
	//}
	//defer resp.Body.Close()
	//
	//bodyBytes, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	panic(err)
	//}
	//
	//body := string(bodyBytes)
	//fmt.Println(body)

	// profiles:
	// block?debug=1
	// goroutine?debug=1
	// heap?debug=1
	// mutex?debug=1
	// threadcreate?debug=1

	// cpu? (?)
	// profile
	// trace

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	objectKeyPrefix := path.Join(defaultObjectPrefix, hostname)

	for {
		url := fmt.Sprintf("%s/%s?debug=%s&seconds=%d", defaultUrl, defaultProfile, defaultDebug, profileDurationSeconds)

		options := &driver.Options{
			Flagset: pprof.NewFlagSet(map[string]string{"proto": "true", "output": "/tmp/out.pb.gz"}, url),
			UI:      &pprof.UI{},
		}
		if err := driver.PProf(options); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}

		err := writeProfileToMinio(client, objectKeyPrefix, "/tmp/out.pb.gz")
		if err != nil {
			panic(err)
		}

		err = cleanupProfilesInMinio(client, objectKeyPrefix, keepCount)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Waiting for %d seconds until next profile\n", profileDelaySeconds)
		time.Sleep(time.Duration(profileDelaySeconds) * time.Second)
	}
}

func cleanupProfilesInMinio(client *minio.Client, objectKeyPrefix string, keepCount int) error {
	doneCh := make(chan struct{}, 1)
	defer close(doneCh)
	var profs []string
	for obj := range client.ListObjects(s3Bucket, objectKeyPrefix, true, doneCh) {
		if obj.Err != nil {
			panic(fmt.Errorf("error reading objects from %s/%s: %v", s3Bucket, defaultObjectPrefix, obj.Err))
		}
		profs = append(profs, obj.Key)
	}

	if keepCount >= len(profs) {
		return nil
	}

	profiles := profiles{profs, objectKeyPrefix}
	sort.Sort(sort.Reverse(profiles))

	deleteByKeepCount := profiles.profs[keepCount:]
	for _, prof := range deleteByKeepCount {
		fmt.Printf("Deleting profile %s in Minio\n", prof)
		err := client.RemoveObject(s3Bucket, prof)
		if err != nil {
			return fmt.Errorf("error deleting profile %s%s", objectKeyPrefix, prof+".pb.gz")
		}
	}
	return nil
}

func writeProfileToMinio(client *minio.Client, objectKeyPrefix, profileFile string) error {
	objectKey := path.Join(objectKeyPrefix, time.Now().Format("15:04:05")+".pb.gz")
	fmt.Printf("Uploading profile to: %s\n", objectKey)
	_, err := client.FPutObject(s3Bucket, objectKey, profileFile, minio.PutObjectOptions{ContentType: "application/zip"})
	return err
}

func initMinio() (*minio.Client, error) {
	client, err := minio.New(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	if err != nil {
		return nil, err
	}

	var transport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          1024,
		MaxIdleConnsPerHost:   1024,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// Set this value so that the underlying transport round-tripper
		// doesn't try to auto decode the body of objects with
		// content-encoding set to `gzip`.
		//
		// Refer:
		//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
		DisableCompression: true,
	}
	client.SetCustomTransport(transport)

	_, err = client.GetBucketLocation(s3Bucket)
	if err != nil {
		return nil, err
	}

	if err != nil {
		// create bucket if it doesn't exist
		if errorResponse, ok := err.(minio.ErrorResponse); ok && errorResponse.Code == "NoSuchBucket" {
			err := client.MakeBucket(s3Bucket, s3Region)
			if err != nil {
				return nil, fmt.Errorf("error creating bucket %s: %v", s3Bucket, err)
			}
		} else {
			return nil, fmt.Errorf("error getting bucket location for bucket %s: %v", s3Bucket, err)
		}
	}
	return client, nil
}

type profiles struct {
	profs        []string
	objectPrefix string
}

func (p profiles) Len() int { return len(p.profs) }
func (p profiles) Less(i, j int) bool {
	iP := strings.TrimSuffix(strings.TrimPrefix(p.profs[i], p.objectPrefix+"/"), ".pb.gz")
	jP := strings.TrimSuffix(strings.TrimPrefix(p.profs[j], p.objectPrefix+"/"), ".pb.gz")

	iT, err := time.Parse("15:04:05", iP)
	if err != nil {
		panic(err)
	}
	jT, err := time.Parse("15:04:05", jP)
	if err != nil {
		panic(err)
	}
	return iT.Before(jT)
}
func (p profiles) Swap(i, j int) { p.profs[i], p.profs[j] = p.profs[j], p.profs[i] }
