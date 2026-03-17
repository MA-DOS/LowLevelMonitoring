package watcher

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/common"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Regex to match Nextflow container names.
var pod_re = regexp.MustCompile(`^nf-[a-f0-9]{32}-[a-f0-9]{5}$`)

// Ensure NextflowPod implements the WorkflowEntity interface
var _ common.WorkflowEntity = (*NextflowPod)(nil)
var mu sync.Mutex

type NextflowPod struct {
	PodEvent  string    `json:"event"`
	StartTime time.Time `json:"start_time"`
	DieTime   time.Time `json:"die_time"`
	Name      string    `json:"name"`
	LifeTime  string    `json:"life_time"`
	PodID     string    `json:"container_id"`
	WorkDir   string    `json:"work_dir"`
}

func InitK8sClient() (*kubernetes.Clientset, error) {

	// The default location for the kubeconfig file is in the user's home directory.
	var kubeconfig string
	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	if kubeconfig == "" {
		err := fmt.Errorf("no kubeconfig present: HOME is not set and kubeconfig path could not be determined")
		fmt.Printf("Error encountered: %v\n", err)
		return nil, err
	}

	if _, err := os.Stat(kubeconfig); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("no kubeconfig present at location %s", kubeconfig)
		} else {
			err = fmt.Errorf("unable to access kubeconfig at %s: %w", kubeconfig, err)
		}
		fmt.Printf("Error encountered: %v\n", err)
		return nil, err
	}

	// Create the client configuration from the kubeconfig file.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Printf("Error encountered: %v\n", err)
		return nil, err
	}

	// Configure client-side rate limiting.
	config.QPS = 50
	config.Burst = 100

	// A clientset contains clients for all the API groups and versions supported
	// by the cluster.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error encountered: %v\n", err)
		return nil, err
	}

	// Use the clientset to interact with the API.
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error encountered: %v\n", err)
		return nil, err
	}
	fmt.Printf("There are %d pods in the default namespace\n", len(pods.Items))
	return clientset, err
}

func (c *NextflowPod) GetPodEvents(client *kubernetes.Clientset, podEventChannel chan<- common.WorkflowEntity) {
	// K8s Client.
	watcher, err := client.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error initializing watcher: %v\n", err)
		return
	}
	eventChannel := watcher.ResultChan()

	processedStarts := make(map[string]bool) // Track started containers
	processedKills := make(map[string]bool)  // Track died containers

	for event := range eventChannel {

		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		if pod_re.MatchString(p.Name) && p.Status.Phase != corev1.PodSucceeded && p.Status.Phase != corev1.PodFailed {
			switch event.Type {
			case watch.Added:
				logrus.Infof("Pod added: %s/%s", p.Namespace, p.Name)
				processPodEvent(p, processedStarts, podEventChannel, true)
			}
			// if pod_re.MatchString(p.Name) {
			// 	switch event.Type {
			// 	case watch.Deleted:
			// 		logrus.Infof("Pod deleted: %s/%s", p.Namespace, p.Name)
			// 		processPodEvent(p, processedKills, podEventChannel, false)
			// 	}
			// }
		} else {
			switch event.Type {
			case watch.Deleted:
				logrus.Infof("Pod deleted: %s/%s", p.Namespace, p.Name)
				processPodEvent(p, processedKills, podEventChannel, false)
			}
		}
	}
}

func processPodEvent(p *corev1.Pod, processed map[string]bool, podEventChannel chan<- common.WorkflowEntity, isStartEvent bool) {
	if len(p.Name) > 0 && pod_re.MatchString(p.Name) {
		processed[p.Name] = true
		logrus.Infof("Found Nextflow pod: %s\n", p.Name)
		nextflowEntity := createNextflowPod(p)

		if isStartEvent {
			// eventType := "[STARTED]"
			if pod, ok := nextflowEntity.(*NextflowPod); ok {
				WriteStartedPodToOutput(*pod)
			}
		}

		if !isStartEvent {
			// eventType = "[DIED]"
			podEventChannel <- nextflowEntity
			if pod, ok := nextflowEntity.(*NextflowPod); ok {
				WriteKilledPodToOutput(*pod)
			}
		}
	}
}

func createNextflowPod(p *corev1.Pod) common.WorkflowEntity {
	var start, kill time.Time
	var lifeTime string

	// Use the pod's StartTime for the start time
	if p.Status.StartTime != nil {
		start = p.Status.StartTime.Time
	}

	// Determine the kill time based on the pod's phase
	if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
		for _, condition := range p.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
				kill = condition.LastTransitionTime.Time
				break
			}
		}
	}

	// Calculate the lifetime if both start and kill times are available
	if !start.IsZero() && !kill.IsZero() {
		lifeTime = fmt.Sprintf("%d ms", kill.Sub(start).Milliseconds())
	}

	return &NextflowPod{
		Name:     p.Name,
		LifeTime: lifeTime,
		PodID:    string(p.UID),
		WorkDir:  p.Labels["workdir"],
	}
}

// Implement WorkflowEntity interface for NextflowPod
func (p NextflowPod) GetStartTime() time.Time {
	return p.StartTime
}

func (p NextflowPod) GetDieTime() time.Time {
	return p.DieTime
}

func (p NextflowPod) GetName() string {
	return p.Name
}

func (p NextflowPod) GetWorkDir() string {
	return p.WorkDir
}

func WriteStartedPodToOutput(pod NextflowPod) {
	fullPath := prepareOutputFile("results", "started_nextflow_pod.csv")
	if fullPath == "" {
		return
	}

	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header if the file is empty
	if isFileEmpty(file) {
		if err := writer.Write([]string{"Name", "PodID", "WorkDir"}); err != nil {
			logrus.Error("Error writing CSV header: ", err)
			return
		}
	}

	// Write pod data to CSV
	if err := writer.Write([]string{
		pod.Name,
		pod.PodID,
		pod.WorkDir,
	}); err != nil {
		logrus.Error("Error writing pod data to CSV: ", err)
	}
}

func WriteKilledPodToOutput(pod NextflowPod) {
	fullPath := preparePodOutputFile("results", "died_nextflow_pods.csv")
	if fullPath == "" {
		return
	}

	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if isPodFileEmpty(file) {
		if err := writer.Write([]string{"Name", "PodID", "LifeTime", "WorkDir"}); err != nil {
			logrus.Error("Error writing CSV header: ", err)
			return
		}
	}

	// Write pod data to CSV
	if err := writer.Write([]string{
		pod.Name,
		pod.PodID,
		pod.LifeTime,
		pod.WorkDir,
	}); err != nil {
		logrus.Error("Error writing pod data to CSV: ", err)
	}
}

func preparePodOutputFile(path, fileName string) string {
	fullPath := fmt.Sprintf("%s/%s", path, fileName)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			logrus.Error("Error creating results directory: ", err)
			return ""
		}
	}

	return fullPath
}

func isPodFileEmpty(file *os.File) bool {
	fileInfo, err := file.Stat()
	if err != nil {
		logrus.Error("Error getting file info: ", err)
		return false
	}
	return fileInfo.Size() == 0
}

func EscapePodName(podName string) string {
	// Remove the leading '/' if present
	podName = strings.TrimPrefix(podName, "/")
	// Escape remaining '/' characters
	return fmt.Sprintf("Cleaned Pod Name for Query: %s", strings.ReplaceAll(podName, "/", `\/`))
}
