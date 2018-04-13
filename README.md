# ElasticSearch Hook for [Logrus](https://github.com/Sirupsen/logrus)
Features:
- Asynchronous
- Updates elastic search in bulks, reducing strain on the ElasticSearch server

## Installation
`go get github.com/interactive-solutions/go-logrus-elasticsearch`

`dep ensure -add github.com/interactive-solutions/go-logrus-elasticsearch`

## Usage
```go
package main
    
import (
    "os"

    "strings"

    "fmt"
    "time"
    
    "github.com/pkg/errors"
    "github.com/sirupsen/logrus"
    "github.com/interactive-solutions/go-logrus-elasticsearch"
    "github.com/olivere/elastic"
)
    
func main() {
    // Create logger
    logger := logrus.New()
    
    // Create elastic client
    client, err := elastic.NewClient(elastic.SetURL("localhost:9200"))
    if err != nil {
        logger.WithError(err).Fatal("Failed to construct elasticsearch client")
    }
    
    // Create logger with 15 seconds flush interval
    hook, err := elastic_logrus.NewElasticHook(client, "some-host", logrus.DebugLevel, func() string {
        return fmt.Sprintf("%s-%s", "some-index", time.Now().Format("2006-01-02"))
    }, time.Second * 15)
    
    if err != nil {
        logger.WithError(err).Fatal("Failed to create elasticsearch hook for logger")
    }
    
    logger.Hooks.Add(hook)
    logger.Info("All done")
}
```
