package main

import (
	"context"
	"flag"
	"github.com/openshift/library-go/pkg/operator/events"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/config/helpers"
)

func main() {
	var kubeConfig string
	klog.InitFlags(flag.CommandLine)
	flag.StringVar(&kubeConfig, "kubeconfig", "", "")
	flag.Parse()

	klog.Info("starting the controllers")
	config, err := helpers.GetKubeConfigOrInClusterConfig(kubeConfig, configv1.ClientConnectionOverrides{})
	if err != nil {
		panic(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	kubeInformers := informers.NewSharedInformerFactory(client, 1*time.Hour)
	memoryRecorder := events.NewInMemoryRecorder("k8s-oc")

	ctrlOne := newControllerOne(client.CoreV1(), kubeInformers.Core().V1(), memoryRecorder)
	ctrlTwo := newControllerTwo(client.CoreV1(), kubeInformers.Core().V1(), memoryRecorder)
	consistencyInvariants := newDataConsistencyInvariants(client.CoreV1())

	ctx := setupSignalContext(context.Background())
	kubeInformers.Start(ctx.Done())
	go ctrlOne.Run(ctx, 1)
	go ctrlTwo.Run(ctx, 1)
	go consistencyInvariants.Run(ctx)

	klog.Info("Waiting for SIGTERM or SIGINT signal to initiate shutdown")
	<-ctx.Done()
}
