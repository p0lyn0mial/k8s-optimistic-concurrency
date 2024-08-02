package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type dataConsistencyInvariants struct {
	client corev1client.CoreV1Interface
}

func newDataConsistencyInvariants(client corev1client.CoreV1Interface) *dataConsistencyInvariants {
	return &dataConsistencyInvariants{
		client: client,
	}
}

func (c *dataConsistencyInvariants) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		w, err := c.client.ConfigMaps("foo").Watch(ctx, metav1.ListOptions{})
		if err != nil {
			klog.Errorf("Failed to watch configmaps: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		err = handleDataConsistencyInvariants(ctx, w)
		if err != nil {
			klog.Error(err)
		}
		w.Stop()
	}
}

func handleDataConsistencyInvariants(ctx context.Context, w watch.Interface) error {
	var observedControllerOneCounter int
	var observedControllerTwoCounter int

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			if event.Type == watch.Error {
				return apierrors.FromObject(event.Object)
			}
			switch event.Type {
			case watch.Modified:
				cm, ok := event.Object.(*v1.ConfigMap)
				if !ok {
					return fmt.Errorf("unexpected object %v, expected a *ConfigMap", event.Object)
				}
				if cm.Name != "bar" {
					continue
				}
				controllerOneCounterStr := cm.Data["controllerOne"]
				controllerTwoCounterStr := cm.Data["controllerTwo"]

				controllerOneCounter, err := strconv.Atoi(strings.TrimPrefix(controllerOneCounterStr, "alive-"))
				if err != nil {
					return err
				}
				controllerTwoCounter, err := strconv.Atoi(strings.TrimPrefix(controllerTwoCounterStr, "alive-"))
				if err != nil {
					return err
				}

				if controllerOneCounter < observedControllerOneCounter {
					return fmt.Errorf("controller one counter invariant not met, controllerOneCounter = %d, observedControllerOneCounter = %d", controllerOneCounter, observedControllerOneCounter)
				}
				if controllerTwoCounter < observedControllerTwoCounter {
					return fmt.Errorf("controller two counter invariant not met, controllerTwoCounter = %d, observedControllerTwoCounter = %d", controllerTwoCounter, observedControllerTwoCounter)
				}
			}
		}
	}
}
