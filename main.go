// Command kube-event-tail connects to a Kubernetes API server and prints out each event that is
// observed.  Like `kubectl get events --all-namespaces --watch`.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jrockway/opinionated-server/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	eventCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubernetes_event_count",
			Help: "A count of events, by namespace",
		},
		[]string{"namespace"},
	)
)

type kflags struct {
	Kubeconfig string `long:"kubeconfig" env:"KUBECONFIG" description:"Kubeconfig to use to connect to the cluster, when running outside of the cluster."`
	Master     string `long:"master" env:"KUBE_MASTER" description:"URL of the kubernetes master, only necessary when running outside of the cluster and when it's not specified in the provided kubeconfig."`
}

func main() {
	server.AppName = "kube-event-tail"

	kf := new(kflags)
	server.AddFlagGroup("Kubernetes", kf)
	server.Setup()

	ctx := context.Background()
	go func() {
		if err := WatchEvents(ctx, kf.Master, kf.Kubeconfig); err != nil {
			zap.L().Fatal("problem watching events", zap.Error(err))
		}
	}()

	server.ListenAndServe()
}

// WatchEvents connects to the k8s API server (using an in-cluster configuration if kubconfig and
// master are empty) and prints out each event that is observed.
func WatchEvents(ctx context.Context, master, kubeconfig string) error {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return fmt.Errorf("kubernetes: build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("kubernetes: new client: %w", err)
	}

	lw := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "events", "", fields.Everything())
	r := cache.NewReflector(lw, &v1.Event{}, &s{}, 0)
	r.Run(ctx.Done())
	return nil
}

func LogEvent(e *v1.Event) {
	if e == nil {
		return
	}
	eventCount.WithLabelValues(e.ObjectMeta.Namespace).Inc()
	msg := "event without message"
	if e.Message != "" {
		msg = e.Message
	}

	objRef := func(o *v1.ObjectReference) zapcore.ObjectMarshaler {
		return zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			if o.Namespace != e.Namespace {
				enc.AddString("namespace", o.Namespace)
			}
			enc.AddString("name", o.Kind+"/"+o.Name)
			if p := o.FieldPath; p != "" {
				enc.AddString("fieldPath", p)
			}
			return nil
		})
	}

	zap.L().Named("event").Info(msg,
		zap.Object("event", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("namespace", e.ObjectMeta.Namespace)
			enc.AddString("name", e.ObjectMeta.Name)
			enc.AddObject("involvedObject", objRef(&e.InvolvedObject))
			if e.Reason != "" {
				enc.AddString("reason", e.Reason)
			}
			src := e.Source
			if src.Component != "" {
				enc.AddString("source.component", src.Component)
			}
			if src.Host != "" {
				enc.AddString("source.host", src.Host)
			}
			if t := e.EventTime.Time; !t.IsZero() {
				enc.AddTime("eventTime", t)
			}
			if t := e.FirstTimestamp.Time; !t.IsZero() {
				enc.AddTime("firstTimestamp", t)
			}
			if t := e.LastTimestamp.Time; !t.IsZero() {
				enc.AddTime("lastTimestamp", t)
			}
			if f, l := e.FirstTimestamp, e.LastTimestamp; !f.IsZero() && !l.IsZero() && l.Sub(f.Time) > time.Second {
				enc.AddDuration("ongoingDuration", l.Sub(f.Time))
			}
			if e.Count > 0 {
				enc.AddInt32("count", e.Count)
			}
			if s := e.Series; s != nil {
				enc.AddInt32("series.count", s.Count)
				enc.AddTime("series.lastObservedTime", s.LastObservedTime.Time)
			}
			if a := e.Action; a != "" {
				enc.AddString("action", a)
			}
			if e.Related != nil {
				enc.AddObject("related", objRef(e.Related))
			}
			if rc := e.ReportingController; rc != "" {
				enc.AddString("reportingController", rc)
			}
			if ri := e.ReportingInstance; ri != "" {
				enc.AddString("reportingInstance", ri)
			}
			if t := e.Type; t != "" && t != "Normal" {
				enc.AddString("type", t)
			}
			return nil
		})),
	)
}

// s is a cache.Store that logs events as they are added.
type s struct{}

// Add implements cache.Store.
func (s *s) Add(obj interface{}) error {
	e, ok := obj.(*v1.Event)
	if !ok {
		return errors.New("non-event object received")
	}
	LogEvent(e)
	return nil
}

// Update implements cache.Store.
//
// NOTE(jrockway): Events are actually updated; this is relevant when looking at fields like
// LastTimestamp and Count.  We can't change logs we've already written, so we treat them as brand
// new events.  What this means is that if you ever wanted to implement a feature like "don't show
// cached events on startup" and suppress events with ObjectMeta.CreationTime before the program
// started up, you'll also suppress events that are happening now but are updates to an event
// resource that was created before the program started.  So some care is necessary, and is why the
// feature currently doesn't exist.
func (s *s) Update(obj interface{}) error {
	return s.Add(obj)
}

// Delete implements cache.Store.
func (*s) Delete(obj interface{}) error { return nil }

// Replace implements cache.Store.
func (s *s) Replace(objs []interface{}, unusedResourceVersion string) error {
	var result error
	for _, obj := range objs {
		if err := s.Add(obj); err != nil {
			result = err
		}
	}
	return result
}

// We only implement cache.Store for cache.Reflector, and cache.Reflector does not call List/Get methods.
func (*s) Resync() error       { return nil }
func (*s) List() []interface{} { return nil }
func (*s) ListKeys() []string  { return nil }
func (*s) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, errors.New("unimplemented")
}
func (*s) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, errors.New("unimplemented")
}
