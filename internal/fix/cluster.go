package fix

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/YunDingLab/fix_log4j2/internal/config"
	"github.com/YunDingLab/fix_log4j2/internal/logs"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	defaultAPIListLimit = 50
)

type kubeCluster struct {
	kcfg *rest.Config
	kcli *kubernetes.Clientset

	vulWorkloads map[string]*kubeWorkload
}

// NewCluster .
func NewCluster() (*kubeCluster, error) {
	kc := &kubeCluster{}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfgfile := config.Conf().MainConf.KubeConfig
		if cfgfile == "" {
			cfgfile = os.Getenv("KUBECONFIG")
		}
		if cfgfile == "" {
			logs.Error("required kube-config file path")
			return nil, fmt.Errorf("required kube-config file path")
		}

		cfg, err = clientcmd.BuildConfigFromFlags("", cfgfile)
		if err != nil {
			logs.Warn("[kube-cluster] got kube-config failed, %s", err)
			return nil, err
		}
		logs.Infof("[kube-cluster] got kube-config (%s) succ", cfgfile)
	} else {
		logs.Infof("[kube-cluster] got inClusterConfig succ")
	}

	kc.kcfg = cfg
	kc.kcli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		logs.Errorf("[kube-cluster] new inCluster k8s client failed, %s", err)
		return nil, err
	}

	return kc, nil
}

func (kc *kubeCluster) RunCheck() error {
	kc.vulWorkloads = map[string]*kubeWorkload{}
	nss, err := kc.ListNamespace()
	if err != nil {
		return err
	}
	podCount := 0
	for _, ns := range nss {
		pods, err := kc.ListPods(ns.Name)
		if err != nil {
			return err
		}
		for _, pod := range pods {
			kw := &kubeWorkload{
				kubeCluster: kc,
				pod:         *pod.DeepCopy(),
			}
			exi, err := kw.Check()
			if err != nil {
				continue
			}
			if !exi {
				continue
			}
			podCount++
			wl := kw.Workload()
			if old, ok := kc.vulWorkloads[wl.ToString()]; ok {
				old.wl.PodNameList = append(old.wl.PodNameList, wl.PodName)
			} else {
				kc.vulWorkloads[wl.ToString()] = kw
			}
		}
	}

	logs.Infof("[kube-cluster] found %v workloads (%v pods) exists vulnerability", len(kc.vulWorkloads), podCount)
	for _, wl := range kc.vulWorkloads {
		if err := wl.Fix(); err != nil {
			logs.Errorw("[fixer] fix failed.",
				"workload", wl.wl.ToString(),
				"error", err)
			return err
		}
		logs.Infow("[fixer] fix succ.", "workload", wl.wl.ToString())
	}
	return nil
}

// ListNamespace list all namespaces
func (kc *kubeCluster) ListNamespace() ([]v1.Namespace, error) {
	var (
		listOpt = metav1.ListOptions{
			Limit: defaultAPIListLimit,
		}
		nslist []v1.Namespace
	)

	for {
		resp, err := kc.retryListNamespaces(3, listOpt)
		if err != nil {
			return nil, err
		}
		if nslist == nil && len(resp.Items) > 0 {
			count := len(resp.Items)
			if resp.RemainingItemCount != nil && *resp.RemainingItemCount > 0 {
				count += int(*resp.RemainingItemCount)
			}
			nslist = make([]v1.Namespace, 0, count)
		}
		nslist = append(nslist, resp.Items...)
		if resp.RemainingItemCount == nil || *resp.RemainingItemCount <= 0 {
			break
		}
		logs.Debugf("[kube-cluster] listContinue continue: %s, Remain: %v", resp.Continue, *resp.RemainingItemCount)
		listOpt.Continue = resp.Continue
	}

	return nslist, nil
}

func (kc *kubeCluster) retryListNamespaces(retryTimes int,
	listOption metav1.ListOptions) (list *v1.NamespaceList, err error) {
	for i := 0; i < retryTimes; i++ {
		list, err = kc.kcli.CoreV1().Namespaces().List(context.Background(), listOption)
		if err != nil {
			logs.Errorf("[kube-cluster][%v] list namespaces failed, %s", i+1, err)
			waitRandomDuration()
			continue
		}
		return list, nil
	}
	return nil, err
}

func (kc *kubeCluster) ListPods(ns string) ([]v1.Pod, error) {
	var (
		listOpt = metav1.ListOptions{
			Limit: defaultAPIListLimit,
		}
		podlist []v1.Pod
	)

	for {
		resp, err := kc.retryListPods(3, ns, listOpt)
		if err != nil {
			return nil, err
		}
		if podlist == nil && len(resp.Items) > 0 {
			count := len(resp.Items)
			if resp.RemainingItemCount != nil && *resp.RemainingItemCount > 0 {
				count += int(*resp.RemainingItemCount)
			}
			podlist = make([]v1.Pod, 0, count)
		}
		podlist = append(podlist, resp.Items...)
		if resp.RemainingItemCount == nil || *resp.RemainingItemCount <= 0 {
			break
		}
		logs.Debugf("[kube-cluster] listContinue continue: %s, Remain: %v", resp.Continue, *resp.RemainingItemCount)
		listOpt.Continue = resp.Continue
	}

	return podlist, nil
}

func (kc *kubeCluster) retryListPods(retryTimes int, namespace string,
	listOption metav1.ListOptions) (list *v1.PodList, err error) {
	for i := 0; i < retryTimes; i++ {
		list, err = kc.kcli.CoreV1().Pods(namespace).List(context.Background(), listOption)
		if err != nil {
			logs.Errorf("[kube-cluster] list pod failed, %s", err)
			waitRandomDuration()
			continue
		}
		return list, nil
	}
	return nil, err
}

// Exec
func (kc *kubeCluster) Exec(namespace, podname, containername string, command []string) (string, error) {
	req := kc.kcli.CoreV1().
		RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podname).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   command,
			Container: containername,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)
	logs.Infof("exec %s", req.URL())
	exec, err := remotecommand.NewSPDYExecutor(kc.kcfg, "POST", req.URL())
	if err != nil {
		return "", err
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	})
	if err != nil {
		return "", err
	}
	if stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf(stderr.String())
	}

	return stdout.String(), nil
}

// GetWorkload 获取pod的创建者信息
// /:namespace/:parent_kind/:parent_name
// parent_kind: DaemonSet/Job/CronJob/Deployment/ReplicaSet/...
func (kc *kubeCluster) GetWorkload(pod v1.Pod) (ret *Workload) {
	ret = getPodRef(pod)

	switch ret.Kind {
	case "Pod":
	case "CronJob":
	case "DaemonSet":
	case "Job":
		if !strings.Contains(ret.Name, "-") {
			return
		}
		job, err := kc.GetJob(ret.Namespace, ret.Name)
		if err != nil {
			logs.Warn("describe job %s/%s failed, %s", ret.Namespace, ret.Name, err)
			return
		}
		updateOwnerReferences(ret, job.ObjectMeta.OwnerReferences)

	case "ReplicaSet":
		if !strings.Contains(ret.Name, "-") {
			return
		}
		rs, err := kc.GetReplicaSet(ret.Namespace, ret.Name)
		if err != nil {
			logs.Warn("describe rs %s/%s failed, %s", ret.Namespace, ret.Name, err)
			return
		}
		updateOwnerReferences(ret, rs.ObjectMeta.OwnerReferences)
	}

	return
}

// GetJob get job
func (kc *kubeCluster) GetJob(namespace, name string) (*batchv1.Job, error) {
	job, err := kc.kcli.BatchV1().Jobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return job, nil
}

// GetReplicaSet get replicaset
func (kc *kubeCluster) GetReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error) {
	rs, err := kc.kcli.AppsV1().ReplicaSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// GetDeployment get Deployment
func (kc *kubeCluster) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	rs, err := kc.kcli.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// GetDaemonSet get DaemonSet
func (kc *kubeCluster) GetDaemonSet(namespace, name string) (*appsv1.DaemonSet, error) {
	rs, err := kc.kcli.AppsV1().DaemonSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// GetStatefulSet get StatefulSet
func (kc *kubeCluster) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	rs, err := kc.kcli.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// UpdateDeployment Deployment
func (kc *kubeCluster) UpdateDeployment(newDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	rs, err := kc.kcli.AppsV1().Deployments(newDeployment.Namespace).
		Update(context.Background(), newDeployment, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// UpdateDaemonSet DaemonSet
func (kc *kubeCluster) UpdateDaemonSet(newDaemonSet *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	rs, err := kc.kcli.AppsV1().DaemonSets(newDaemonSet.Namespace).
		Update(context.Background(), newDaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// UpdateStatefulSet StatefulSet
func (kc *kubeCluster) UpdateStatefulSet(newStatefulSet *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	rs, err := kc.kcli.AppsV1().StatefulSets(newStatefulSet.Namespace).
		Update(context.Background(), newStatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// getPodRef 获取pod的上层信息
func getPodRef(pod v1.Pod) *Workload {
	wl := &Workload{
		Namespace:     pod.Namespace,
		PodName:       pod.Name,
		PodNameList:   []string{pod.Name},
		Kind:          "Pod",
		Name:          pod.Name,
		ContainerInfo: map[string]string{},
	}
	updateOwnerReferences(wl, pod.ObjectMeta.OwnerReferences)
	return wl
}

func updateOwnerReferences(wl *Workload, refs []metav1.OwnerReference) {
	for _, ref := range refs {
		if ref.Kind == "Node" {
			continue
		}
		wl.Kind = ref.Kind
		wl.Name = ref.Name
		return
	}
}

// waitRandomDuration wait 1~5 seconds
func waitRandomDuration() {
	randSec := rand.Int63n(5)
	dur := time.Duration(1+randSec) * time.Second
	time.Sleep(dur)
}

// Workload workload info
type Workload struct {
	Namespace  string `json:"ns,omitempty"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	VerifyInfo string `json:"verify_info,omitempty"`

	// PodName Pod的名称，不用于上报
	PodName     string   `json:"-"`
	PodNameList []string `json:"-"`

	ContainerInfo map[string]string
}

// IsEqual ==
func (w Workload) IsEqual(a Workload) bool {
	return w.Kind == a.Kind && w.Namespace == a.Namespace && w.Name == a.Name
}

// WorkloadList list
type WorkloadList []*Workload

// ToString format to string
func (w Workload) ToString() string {
	return fmt.Sprintf("/%s/%s/%s", w.Kind, w.Namespace, w.Name)
}
