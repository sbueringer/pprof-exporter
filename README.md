
# Usage

````bash
# Start exporter with
go run cmd/pprof-exporter/main.go -s3-endpoint=${endpoint} -s3-access-key-id=${access-key-id} -s3-secret-access-key=${secret-access-key} -s3-region=${region} -s3-bucket=${bucket}

# Start importer with:
go run cmd/pprof-importer/main.go -s3-endpoint=${endpoint} -s3-access-key-id=${access-key-id} -s3-secret-access-key=${secret-access-key} -s3-region=${region} -s3-bucket=${bucket}

````

# TODO

* parameterize both completely
* check which types of profiles makes sense (chressie?)
* create aliases

* Implement pprof-exporter container (drop-in to nodeexporter)
    * if possible get node_exporter binary from somewhere and put it in the profile
        * maybe via patching nodexporter image via a predefined way 
        * then deploy node_exporter and node_exporter+pprof-exporter in the same image
