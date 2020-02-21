package main

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This exists to check that nothing panics under strange circumstances.
func TestLogs(t *testing.T) {
	l := zaptest.NewLogger(t)
	zap.ReplaceGlobals(l)

	var nothing *v1.Event
	testData := []interface{}{
		nil,         // nothing
		nothing,     // typed nothing
		&v1.Node{},  // wrong type
		&v1.Event{}, // very empty
		&v1.Event{ // somewhat empty
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
		},
		&v1.Event{ // full
			TypeMeta: metav1.TypeMeta{
				Kind:       "Event",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
			Related: &v1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "foo",
				Name:       "bar",
				FieldPath:  "spec.containers{quux}",
			},
			Series: &v1.EventSeries{
				Count:            42,
				LastObservedTime: metav1.NewMicroTime(time.Now()),
				State:            v1.EventSeriesStateOngoing,
			},
			Source: v1.EventSource{
				Host:      "localhost",
				Component: "kubelet",
			},
			EventTime:           metav1.NewMicroTime(time.Now()),
			FirstTimestamp:      metav1.NewTime(time.Now().Add(-time.Hour)),
			LastTimestamp:       metav1.NewTime(time.Now()),
			Count:               42,
			Type:                "Warning",
			ReportingInstance:   "ri",
			ReportingController: "rc",
			Message:             "Error: OH NOES!",
			Reason:              "reason",
			Action:              "action",
		},
		&v1.Event{ // different namespaces
			ObjectMeta: metav1.ObjectMeta{Namespace: "a"},
			Related:    &v1.ObjectReference{Namespace: "b"},
		},
	}

	s := new(s)
	for _, input := range testData {
		s.Add(input)
		s.Delete(input)
		s.Get(input)
		s.GetByKey("bar")
		s.List()
		s.ListKeys()
		s.Replace([]interface{}{input}, "")
		s.Update(input)
		s.Resync()
	}
}
