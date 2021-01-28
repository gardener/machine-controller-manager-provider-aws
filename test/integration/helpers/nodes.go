package helpers

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

//ProbeNodes tries to probe for nodes. Indirectly it checks whether the cluster is accessible.
// If not accessible, then it returns an error
func (c *Cluster) ProbeNodes() error {
	_, err := c.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	return err
}

//getNodes tries to retrieve the list of node objects in the cluster.
func (c *Cluster) getNodes() (*v1.NodeList, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	return nodes, err
}

//WaitForNodeEvent waits until a given event is generated.
//INProgress
func (c *Cluster) WaitForNodeEvent(eventType watch.EventType, timeOut int64) error {
	w, err := c.clientset.CoreV1().Nodes().Watch(metav1.ListOptions{
		TimeoutSeconds: &timeOut,
	})
	for event := range w.ResultChan() {
		//fmt.Printf("Type: %v\n", event.Type)
		n, ok := event.Object.(*v1.Node)
		if ok {
			text, _ := json.Marshal(n)
			fmt.Printf("Output: %s", string(text))
		}
		if event.Type == "ADDED" {
			//break
			fmt.Println("ADDED")
		}
	}
	return err
}

//NumberOfReadyNodes tries to retrieve the list of node objects in the cluster.
func (c *Cluster) NumberOfReadyNodes() int16 {
	nodes, err := c.getNodes()
	if err != nil {
		panic("Get nodes failed")
	}
	count := int16(0)
	for _, n := range nodes.Items {
		for _, c := range n.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				count++
			}
		}
	}
	return count
}

//NumberOfNodes tries to retrieve the list of node objects in the cluster.
func (c *Cluster) NumberOfNodes() int16 {
	nodes, err := c.getNodes()
	if err != nil {
		panic("Get nodes failed")
	}
	return int16(len(nodes.Items))
}
