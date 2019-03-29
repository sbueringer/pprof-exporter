package main

import (
	"flag"
	"fmt"
	"github.com/google/pprof/driver"
	"github.com/minio/minio-go"
	"github.com/sbueringer/pprof-exporter/pkg/pprof"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

var defaultAddr = "localhost:4000"
var defaultObjectPrefix = "profiles/"

var s3Endpoint string
var s3AccessKeyID string
var s3SecretAccessKey string
var s3Region string
var s3Bucket string

var overviewPageTemplate = `
<html>
<head>
	<title>Profiles</title>
</head>
<body>
	{{ range $podName, $profiles := .PodProfiles }}
	<h1>{{ $podName }}</h1>
	{{ range $podName, $profile := $profiles }}
	<a href="./{{ $profile.Link }}">{{ $profile.LinkText}}</a></br>
	{{- end }}
	{{- end }}
</body>	
</html>
`

func init() {
	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "s3 Endpoint.")
	flag.StringVar(&s3AccessKeyID, "s3-access-key-id", os.Getenv("S3_ACCESS_KEY_ID"), "s3 Access Key ID.")
	flag.StringVar(&s3SecretAccessKey, "s3-secret-access-key", os.Getenv("S3_SECRET_ACCESS_KEY"), "s3 Secret Access Key.")
	flag.StringVar(&s3Region, "s3-region", "", "s3 Region.")
	flag.StringVar(&s3Bucket, "s3-bucket", "backup", "s3 Bucket.")
}

func main() {
	flag.Parse()

	client, err := minio.New(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	if err != nil {
		panic(err)
	}

	_, err = client.GetBucketLocation(s3Bucket)
	if err != nil {
		panic(err)
	}

	doneCh := make(chan struct{}, 1)
	defer close(doneCh)
	var objInfos []minio.ObjectInfo
	for obj := range client.ListObjects(s3Bucket, defaultObjectPrefix, true, doneCh) {
		if obj.Err != nil {
			panic(fmt.Errorf("error reading objects from %s/%s: %v", s3Bucket, defaultObjectPrefix, obj.Err))
		}
		objInfos = append(objInfos, obj)
	}

	mux := http.NewServeMux()

	data := convertToData(objInfos)
	o := overviewPage{Data: data}
	mux.Handle("/", o)

	for i, objInfo := range objInfos {
		object, err := client.GetObject(s3Bucket, objInfo.Key, minio.GetObjectOptions{})
		if err != nil {
			panic(err)
		}

		tmpFile, err := ioutil.TempFile("", "pprof-")
		if err != nil {
			panic(err)
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, object)
		if err != nil {
			panic(er1r)
		}
		addHandlerToMux(mux, i, tmpFile.Name())
	}

	fmt.Println("Listening on localhost:4000")
	if err := http.ListenAndServe("localhost:4000", mux); err != nil {
		panic(err)
	}
}

func convertToData(objInfos []minio.ObjectInfo) Data {
	data := Data{}
	data.PodProfiles = map[string][]PodProfile{}
	regex := regexp.MustCompile("profiles/(.*)/(.*)")
	for i, objInfo := range objInfos {
		matches := regex.FindAllStringSubmatch(objInfo.Key, -1)

		if len(matches) == 1 && len(matches[0]) == 3 {
			podName := matches[0][1]
			profileName := matches[0][2]

			pp, ok := data.PodProfiles[podName]
			if !ok {
				pp = []PodProfile{}
			}
			pp = append(pp, PodProfile{fmt.Sprintf("/%d", i), profileName})
			data.PodProfiles[podName] = pp
		}
	}
	return data
}

type overviewPage struct {
	Data Data
}

func (o overviewPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t := template.New("overview")
	t, _ = t.Parse(overviewPageTemplate)

	t.Execute(w, o.Data)
}

func addHandlerToMux(mux *http.ServeMux, index int, file string) {
	fmt.Printf("Adding profile for file: %s\n", file)
	options := &driver.Options{
		Flagset: pprof.NewFlagSet(map[string]string{"http": "localhost:4000"}, file),
		HTTPServer: func(args *driver.HTTPServerArgs) error {
			for path, handler := range args.Handlers {
				mux.Handle(fmt.Sprintf("/%d%s", index, path), handler)
			}
			return nil
		},
		UI: &pprof.UI{},
	}
	if err := driver.PProf(options); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
}

type Data struct {
	PodProfiles map[string][]PodProfile
}

type PodProfile struct {
	Link     string
	LinkText string
}
