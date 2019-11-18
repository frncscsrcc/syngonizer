SynGoNizer
===

[![Go Report Card](https://goreportcard.com/badge/github.com/frncscsrcc/syngonizer)](https://goreportcard.com/report/github.com/frncscsrcc/syngonizer)

Watch and sync files from your local env to one or more running containers on
a Kubernetes infrastructure.

*Note*: This is a WiP. This first version uses _kubectl_ to connect to the
running kubernetes cluster. In future it will be replaced with a better implementation,
based on the official kubernetes client API. But this is enough for a raining Sunday.

Usage
---
```
  go get github.com/frncscsrcc/syngonizer
  go build src/github.com/frncscsrcc/syngonizer/cmd/syngonizer.go
  ./syngonizer.go configuration.json
````

Config example
---
```
{
  "global": {
    // Events watch time (sec)
    "event-listen-iterval": 0.50,
    // How often get an updated list of pods for the required namespace (sec)
    "update-pod-list-interval": 20,
    // Kube namespace
    "namespace": "default",
    // kubectl bin
    "kubectl-path": "/snap/bin/microk8s.kubectl",
    // Allow to sync if namespace = "production"
    "allow-production": false,
    // Die in case of errors
    "die-if-error": false
  },
  "folders": [
    {
      // Absolute path for a watched folder
      "local-root" : "/home/project/appACode/",
      // Remote path on the container (optional)
      "remote-root" : "/container/folder/",
      // "app" label to be used (better selector in next version)
      "apps" : ["appA1", "appA2"]
    },
    {
      "local-root" : "/home/project/commonCode/",
      "remote-root" : "/container/folder/",
      "apps" : ["appA1", "appA2", "appB1"]
    },
    {
      "local-root" : "/home/project/appBCode/",
      "apps" : ["appB1"]
    }    
  ]
}
```


Documentation, configuration examples and hopefully unit tests will be added ASAP.

Made with <3 from and enthusiastic (and still naive) Go developer.
