package elastic_logrus

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"gopkg.in/olivere/elastic.v5"
)

var (
	// Fired if the
	// index is not created
	ErrCannotCreateIndex = fmt.Errorf("Cannot create index")
)

type IndexNameFunc func() string

type ElasticSearchHook struct {
	processor *elastic.BulkProcessor
	host      string
	index     IndexNameFunc
	levels    []logrus.Level
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewElasticHook(
	client *elastic.Client,
	host string,
	level logrus.Level,
	indexFunc IndexNameFunc,
	flushInterval time.Duration,
) (*ElasticSearchHook, error) {

	levels := []logrus.Level{}
	for _, l := range []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	} {
		if l <= level {
			levels = append(levels, l)
		}
	}

	ctx, cancel := context.WithCancel(context.TODO())

	exists, err := client.IndexExists(indexFunc()).Do(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		createIndex, err := client.CreateIndex(indexFunc()).Do(ctx)
		if err != nil {
			return nil, err
		}
		if !createIndex.Acknowledged {
			return nil, ErrCannotCreateIndex
		}
	}

	// from elastic docs: "If you want the bulk processor to
	// operate completely asynchronously, set both BulkActions and BulkSize to
	// -1 and set the FlushInterval to a meaningful flushInterval."
	processor, err := client.BulkProcessor().
		Name(host).
		BulkActions(-1).
		BulkSize(-1).
		FlushInterval(flushInterval).
		Do(ctx)

	if err != nil {
		return nil, errors.Wrap(err, "Unable to create bulk processor")
	}

	return &ElasticSearchHook{
		processor: processor,
		index:     indexFunc,
		host:      host,
		levels:    levels,
		ctx:       ctx,
		ctxCancel: cancel,
	}, nil
}

func (hook *ElasticSearchHook) Fire(entry *logrus.Entry) error {
	level := entry.Level.String()

	if e, ok := entry.Data[logrus.ErrorKey]; ok && e != nil {
		if err, ok := e.(error); ok {
			entry.Data[logrus.ErrorKey] = err.Error()
		}
	}

	msg := struct {
		Host      string
		Timestamp string `json:"@timestamp"`
		Message   string
		Data      logrus.Fields
		Level     string
	}{
		hook.host,
		entry.Time.UTC().Format(time.RFC3339Nano),
		entry.Message,
		entry.Data,
		strings.ToUpper(level),
	}

	r := elastic.NewBulkIndexRequest().Index(hook.index()).Type("log").Doc(msg)
	hook.processor.Add(r)

	return nil
}

func (hook *ElasticSearchHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *ElasticSearchHook) Cancel() {
	hook.ctxCancel()
}
