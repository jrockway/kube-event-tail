# kube-event-tail

Have you ever noticed some issue in your cluster, went to run `kubectl get events`, and noticed that
Kubernetes helpfully deleted all the relevant events? This program is like running
`kubectl get events --watch`, logging all those events to a log file that you can retain just like a
normal application log file.

The log is formatted as JSON (via zap), and sticks to the same names as the `v1.Event` object. It
omits empty fields and includes a few generated fields to make the logs easier to read for humans.
It also exports an event count, by namespace, for Prometheus to scrape. Then you can be made aware
of an unusually large number of events.

Try it locally:

    $ go get github.com/jrockway/kube-event-tail
    $ kube-event-tail --kubeconfig ~/.kube/config
    {"level":"info","ts":1582326064.0164192,"logger":"event","caller":"kube-event-tail/main.go:97","msg":"Successfully assigned default/busybox to cpus-dcs0","event":{"namespace":"default","name":"busybox.15f58b292ad70619","involvedObject":{"name":"Pod/busybox"},"reason":"Scheduled","source.component":"default-scheduler","eventTime":1582323853.019242,"action":"Binding","reportingController":"default-scheduler","reportingInstance":"default-scheduler-master-jrockus"}}
    ... forever ...

Inspect the metrics:

    $ curl -s localhost:8081/metrics | grep kubernetes_event_count
    # HELP kubernetes_event_count A count of events, by namespace
    # TYPE kubernetes_event_count counter
    kubernetes_event_count{namespace="default"} 7
    kubernetes_event_count{namespace="foo"} 42

Install into your cluster:

    $ kubectl apply -k github.com/jrockway/kube-event-tail/deploy

No additional configuration is required, or available. (It creates a deployment in the `kube-system`
namespace with one replica, and sets up the necessary RBAC machinery to be able to watch the
events.) After deploying, you should be able to
`kubectl logs -n kube-system kube-event-tail-xxxxxxxxxx-yyyyy` and see events.

Never be disappointed that your events went missing before you had time to investigate; now they're
with the rest of your logs!
