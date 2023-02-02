# tekton-s3-log-reader

The aim of this project is to read long term logs used by Tekton Dashboard so we will be able to release resources in your Kubernetes Cluster.

## Architecture
![architecture](img/tekton-logs-arch.png)

## Deploy tekton-s3-log-reader

This application needs [AWS SDK Authentication](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials) to run. In our case, we use a ServiceAccount with `eks.amazonaws.com/role-arn` annotation to authenticate with AWS services from EKS cluster.

In addition, you can use `S3_BUCKET_NAME` where the application will find the logs or one of the following arguments and `AWS_REGION` to get the session:

```bash
Usage of ./tekton-s3-log-reader:
  -b string
    	Bucket name or S3_BUCKET_NAME env var
  -cert string
    	TLS Certificate
  -containerd
    	Use containerd log format
  -key string
    	TLS Key
  -p string
    	Port (default ":5001")
  -r string
    	AWS Region or AWS_REGION env var
  -tls
    	Use TLS to expose endpoint
```

## Observability
tekton-s3-log-reader has `/metrics` endpoint to monitor the behaviour using Prometheus.

## Sample stack configuration
### FluentBit Configuration
#### Docker
```yaml
customParsers: |
    [PARSER]
        Name         docker-custom
        Format       json
        Time_Key     time
        Time_Format  %Y-%m-%dT%H:%M:%S.%L
        Time_Keep Off
        json_date_key false
        # Command      |  Decoder | Field | Optional Action
        # =============|==================|=================
        Decode_Field_As   escaped_utf8    log    do_next
        Decode_Field_As   json       log
filters: |
    [FILTER]
        Name record_modifier
        Match tekton.*
        Remove_key stream
        Remove_key stdout
        Remove_key time
        Remove_key date
input: |
    [INPUT]
        Name              tail
        Alias             tekton-semantic
        Path              /var/log/containers/*_build-release_*
        Parser            docker-custom
        Tag               tekton.<namespace_name>.<pod_name>.<container_name>
        Tag_Regex         (?<pod_name>[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)_(?<namespace_name>[^_]+)_(?<container_name>.+)-
        Mem_Buf_Limit     100MB
        Refresh_Interval  60
outputs: |
    [OUTPUT]
        Name            s3
        Alias           s3_tekton_logs
        Match           tekton.*
        bucket          YOUR_BUCKET
        region          eu-central-1
        total_file_size 250M
        upload_timeout  1m
        s3_key_format   /$TAG[1]/$TAG[2]/$TAG[3]/%Y%m%d%H%M%S.log
        s3_key_format_tag_delimiters .
```
#### Containerd
```yaml
customParsers: |
    [PARSER]
        # http://rubular.com/r/tjUt3Awgg4
        Name cri-custom
        Format regex
        Regex ^(?<time>[^ ]+) (?<stream>stdout|stderr) (?<logtag>[^ ]*) (?<log>.*)$
        Time_Key    time
        Time_Format %Y-%m-%dT%H:%M:%S.%L%z
filters: |
    [FILTER]
        Name record_modifier
        Match tekton.*
        Remove_key logtag
        Remove_key stream
inputs: |
    [INPUT]
        Name              tail
        Alias             tekton-semantic
        Path              /var/log/containers/*_build-release_*
        Parser            cri-custom
        Tag               tekton.<namespace_name>.<pod_name>.<container_name>
        Tag_Regex         (?<pod_name>[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)_(?<namespace_name>[^_]+)_(?<container_name>.+)-
        Mem_Buf_Limit     100MB
        Refresh_Interval  60
        #DB                /fluentbit/db/tail.tekton.db
        # the database is accessed only by Fluent Bit
        #DB.locking        True
        # Skip_Long_Lines On
outputs: |
    
    [OUTPUT]
        Name            s3
        Alias           s3_tekton_logs
        Match           tekton.*
        bucket          YOUR_BUCKET
        region          eu-central-1
        total_file_size 250M
        upload_timeout  1m
        s3_key_format   /$TAG[1]/$TAG[2]/$TAG[3]/%Y%m%d%H%M%S.log
        s3_key_format_tag_delimiters .
```
### Tekton Dashboard Configuration

Add the following flag pointing to the endpoint `tekton-s3-log-reader`: 
```yaml
--external-logs=http://tekton-s3-log-reader.monitoring:5001/logs
```
