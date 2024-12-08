# dkube

To crawl all the static information from a specific namespace in Kubernetes. 

- Get all unique Docker images.
- List all actively used resources.
- Crawl the yaml from each active resources.
- Store all the above information in a directory structure.

```
-- namespace
   docker_images.txt
   active_resources.txt
   -- resource 1
      manifest_resource_name.yaml
   -- resource 2
      manifest_resource_name.yaml
```

- This acts as a snapshot of the environment.
- You can feed the results to perform SAST scan using [Kube-linter](https://github.com/stackrox/kube-linter), [kube-score](https://github.com/zegl/kube-score), [checkov](https://github.com/bridgecrewio/checkov), etc.
- Also, using [docker-multi-scan](https://github.com/okpalindrome/docker-multi-scan) you can scan the images using grype, trivy and docker-scout at once.


### Installation
```
go install -v github.com/okpalindrome/dkube@latest
```

### Usage
```
$ go run main.go --help
Usage of dkube:
  -destination string
        Destination directory/folder to save.
  -namespace string
        Provide namespace.
```

### Note
Used kubectl instead of the [client-go API](https://github.com/kubernetes/client-go?tab=readme-ov-file#compatibility-matrix) because, as a pentester, it is unlikely that I will consistently have access to the same version. 
