
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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/signal"
	"os"
	"syscall"
	"k8s.io/client-go/kubernetes"
	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"fmt"
)

//cron controller struct
type CronJobController struct {
	client    kubernetes.Interface
	logger       *logrus.Entry
}

func main() {
	
	//get config
	kubeClient := GetClientOutOfCluster()
	//create controller
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
		client:	kubeClient,
		logger:       logrus.WithField("pkg", "kubewatch-deployment"),
	}

	return jm
}



func (jm *CronJobController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Infof("Starting CronJob Manager")
	//Every 10 seconds run jm.syncAll()
	go wait.Until(jm.syncAll, 10*time.Second, stopCh)
	<-stopCh
	glog.Infof("Shutting down CronJob Manager")
}

func (jm *CronJobController) syncAll() {
	jm.logger.Infof("Time Now %s",time.Now().Format(time.RFC3339))
	//get all deployments
	deploymentList, err := jm.client.ExtensionsV1beta1().Deployments(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("can't list Jobs: %v", err))
		return
	}
	//go through all deploymenst and see if they have start and stop times 
	deployments := deploymentList.Items
	jm.logger.Infof("Found %d deployments",len(deployments))
        for _, deployment := range deployments {
		startTimeString, boolStartTime := deployment.ObjectMeta.Annotations["startTime"]
		stopTimeString, boolStopTime := deployment.ObjectMeta.Annotations["stopTime"]
		if boolStartTime && boolStopTime   {
			jm.logger.Infof("Start Time %s",startTimeString)
			jm.logger.Infof("Stop Time %s", stopTimeString)

			startTime, err := time.Parse(time.RFC822,startTimeString)
			if err != nil{
				utilruntime.HandleError(fmt.Errorf("Error Parsing start time %s",startTimeString))
			}

			stopTime, err := time.Parse(time.RFC822, stopTimeString)
			if err != nil{
				utilruntime.HandleError(fmt.Errorf("Error Parsing stop time %s",stopTimeString))
			}

			if inTimeSpan(startTime, stopTime, time.Now()){
				jm.logger.Infof("IN. Should be on!")
				if *deployment.Spec.Replicas > int32(0) {
					jm.logger.Infof("Replicas are greater than 0 and thats is CORRECT")
				} else {
					jm.logger.Infof("Replicas are 0 and should increased to 1 CHANGE")
					*deployment.Spec.Replicas = 1
					_, err := jm.client.ExtensionsV1beta1().Deployments(deployment.ObjectMeta.Namespace).Update(&deployment)
					if err != nil{
						utilruntime.HandleError(fmt.Errorf("Error updating deployment: %v",err))
					}
				}
			} else {
				jm.logger.Infof("OUT. should be off!")
				if *deployment.Spec.Replicas > int32(0) {
					jm.logger.Infof("Replicas are greater than 1 and should be decreased to 0 CHANGE")
					*deployment.Spec.Replicas = 0
					_, err := jm.client.ExtensionsV1beta1().Deployments(deployment.ObjectMeta.Namespace).Update(&deployment)
					if err != nil{
						utilruntime.HandleError(fmt.Errorf("Error updating deployment: %v",err))
					}
				} else {
					jm.logger.Infof("Replicas are 0 and that is CORRECT")
				}
			}
		}
	}
}

func inTimeSpan(start time.Time, end time.Time, check time.Time) bool {
	return check.After(start) && check.Before(end)
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
