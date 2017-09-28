
package main

import (

	"time"

	"github.com/golang/glog"

	//"k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	//clientset "k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/kubernetes/scheme"
	//v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	//"k8s.io/client-go/tools/record"
	//"k8s.io/kubernetes/pkg/util/metrics"
	"os/signal"
	"os"
	"syscall"
	"k8s.io/client-go/kubernetes"
	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

)


type CronJobController struct {
	clientset    kubernetes.Interface
	logger       *logrus.Entry
}

func main() {
	kubeClient := GetClientOutOfCluster()

	c := NewCronJobController(kubeClient)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func NewCronJobController(kubeClient kubernetes.Interface) *CronJobController {
	jm := &CronJobController{
		clientset:	kubeClient,
		logger:       logrus.WithField("pkg", "kubewatch-deployment"),
	}

	return jm
}



func (jm *CronJobController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Infof("Starting CronJob Manager")
	// Check things every 10 second.
	go wait.Until(jm.syncAll, 10*time.Second, stopCh)
	<-stopCh
	glog.Infof("Shutting down CronJob Manager")
}

func (jm *CronJobController) syncAll() {
	jm.logger.Infof("Time Now %s",time.Now().Format(time.RFC3339))
}

func GetClientOutOfCluster() kubernetes.Interface {
	config, err := buildOutOfClusterConfig()
	if err != nil {
		logrus.Fatalf("Can not get kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	return clientset
}


func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}
