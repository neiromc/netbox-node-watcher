package main

import (
	"context"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	StatusNetBoxActive      NetboxStatus = "Online"
	StatusNetBoxMaintenance NetboxStatus = "Maintenance mode"
)

type NetboxStatus string

type NodeState struct {
	name         string
	netboxStatus NetboxStatus
}

var (
	config, _    = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	clientSet, _ = kubernetes.NewForConfig(config)
)

func watchNodes() {
	timeOut := int64(60)
	watcher, _ := clientSet.CoreV1().Nodes().Watch(context.Background(), metav1.ListOptions{TimeoutSeconds: &timeOut})

	for event := range watcher.ResultChan() {
		node := event.Object.(*corev1.Node)

		switch event.Type {
		case watch.Modified:
			processNode(node)

		}
	}
}

func processNode(node *corev1.Node) {
	log.Info("Some processing for node: ", node.GetName())
	log.Info("Node is cordoned: ", node.Spec.Unschedulable)
	nodeIsReady := false
	for _, nodeCondition := range node.Status.Conditions {
		if nodeCondition.Type == corev1.NodeReady &&
			nodeCondition.Status == corev1.ConditionTrue {
			nodeIsReady = true
		}
	}
	log.Info("Node is ready: ", nodeIsReady)

	//
	// Select nodeState Netbox status
	//
	nodeState := NodeState{
		node.GetName(),
		nodeState(node.Spec.Unschedulable, nodeIsReady),
	}

	//
	// Send REQ to Netbox
	//
	reportToNetbox(nodeState)

}

func nodeState(stateCordon, stateReady bool) NetboxStatus {
	if !stateCordon && stateReady {
		return StatusNetBoxActive
	}

	return StatusNetBoxMaintenance
}

func reportToNetbox(nodeState NodeState) {
	// http get/post of netbox api url
	log.Infof("===> Netbox hadler. Host=%s nodeState=%s", nodeState.name, nodeState.netboxStatus)
}

func main() {
	var wg sync.WaitGroup
	go watchNodes()
	wg.Add(1)
	wg.Wait()
}
