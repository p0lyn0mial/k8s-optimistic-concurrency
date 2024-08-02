package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

type controllerTwo struct {
	updateCounter   int
	client          corev1client.CoreV1Interface
	configMapLister corev1listers.ConfigMapLister
}

func newControllerTwo(
	client corev1client.CoreV1Interface,
	coreInformers corev1informers.Interface,
	recorder events.Recorder,
) factory.Controller {
	c := &controllerTwo{
		client:          client,
		configMapLister: coreInformers.ConfigMaps().Lister(),
	}

	return factory.New().
		WithInformers(coreInformers.ConfigMaps().Informer()).
		WithSync(c.sync).ResyncEvery(10*time.Second).
		ToController("controllerTwo", recorder.WithComponentSuffix("controller-two"))
}

func (c *controllerTwo) sync(ctx context.Context, _ factory.SyncContext) error {
	configMap, err := c.configMapLister.ConfigMaps("foo").Get("bar")
	if err != nil {
		return err
	}
	configMapCopy := configMap.DeepCopy()
	configMapCopy.Data["controllerTwo"] = fmt.Sprintf("alive-%d", c.updateCounter)

	_, err = c.client.ConfigMaps("foo").Update(ctx, configMapCopy, metav1.UpdateOptions{})
	c.updateCounter++
	return err
}
