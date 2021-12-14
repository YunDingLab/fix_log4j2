package fix

import (
	"fmt"
	"strings"
	"time"

	"github.com/YunDingLab/fix_log4j2/internal/config"
	"github.com/YunDingLab/fix_log4j2/internal/logs"
	v1 "k8s.io/api/core/v1"
)

var checkContainImages map[string]struct{}

type kubeWorkload struct {
	*kubeCluster

	pod v1.Pod
	wl  *Workload
}

func (kw *kubeWorkload) Workload() *Workload {
	return kw.wl
}

func (kw *kubeWorkload) Check() (exi bool, err error) {
	if checkContainImages == nil {
		checkContainImages = map[string]struct{}{}
		for _, v := range config.Conf().Clue.Images {
			checkContainImages[v] = struct{}{}
		}
	}
	for i, cons := range kw.pod.Status.ContainerStatuses {
		if _, ok := checkContainImages[cons.Image]; ok {
			exi = true
			if kw.wl == nil {
				kw.wl = kw.GetWorkload(kw.pod)
			}
			kw.wl.ContainerInfo[cons.Name] = cons.Image
			logs.Infow("[checker] found vulnerability workload",
				"workload", kw.wl.ToString(),
				"pod", kw.wl.PodName,
				"container", cons.Name,
				"image", cons.Image)
			specCon := kw.pod.Spec.Containers[i]
			if len(specCon.Command) == 0 && len(specCon.Args) == 0 {
				kw.pod.Spec.Containers[i].Command, err = kw.getCommandInContainer(cons.Name)
				if err != nil {
					logs.Errorf("exec failed, %s", err)
					return false, err
				}
				logs.Infof("got commands by exec in container, %v", specCon.Command)
			}
			continue
		}
		if _, ok := checkContainImages[cons.ImageID]; ok {
			exi = true
			if kw.wl == nil {
				kw.wl = kw.GetWorkload(kw.pod)
			}
			kw.wl.ContainerInfo[cons.Name] = cons.ImageID
			logs.Infow("[checker] found vulnerability workload",
				"workload", kw.wl.ToString(),
				"pod", kw.wl.PodName,
				"container", cons.Name,
				"image_id", cons.ImageID)
			specCon := kw.pod.Spec.Containers[i]
			if len(specCon.Command) == 0 && len(specCon.Args) == 0 {
				kw.pod.Spec.Containers[i].Command, err = kw.getCommandInContainer(cons.Name)
				if err != nil {
					logs.Errorf("exec failed, %s", err)
					return false, err
				}
				logs.Infof("got commands by exec in container, %v", specCon.Command)
			}
			continue
		}
	}

	return
}

func (kw *kubeWorkload) getCommandInContainer(containerName string) ([]string, error) {
	pscmd := []string{"ps", "-e", "-o", "args="}
	body, err := kw.Exec(kw.pod.Namespace, kw.pod.Name, containerName, pscmd)
	if err != nil {
		logs.Errorf("exec failed, %s", err)
		return nil, err
	}
	lines := strings.Split(body, "\n")

	cmdLine := 0
	cmd := ""
	for _, line := range lines {
		if line == "" {
			continue
		}
		if line == "ps -e -o args=" {
			continue
		}
		if line == "bash" {
			continue
		}
		if line == "sh" {
			continue
		}
		if line == "zsh" {
			continue
		}
		cmdLine++
		cmd = line
		if cmdLine > 1 {
			return nil, fmt.Errorf("not found container command for start")
		}
	}
	return strings.Split(cmd, " "), nil
}

func (kw *kubeWorkload) Fix() error {
	switch kw.wl.Kind {
	case "Deployment":
		return kw.fixDeployment()
	case "DaemonSet":
		return kw.fixDaemonSet()
	case "StatefulSet":
		return kw.fixStatefulSet()
	case "ReplicaSet":
	case "Job":
	case "CronJob":
	case "ReplicationController":
	}
	logs.Errorf("[fixer] not support workload type %s", kw.wl.Kind)
	return nil
}

func (kw *kubeWorkload) fixDeployment() error {
	dep, err := kw.kubeCluster.GetDeployment(kw.wl.Namespace, kw.wl.Name)
	if err != nil {
		logs.Errorw("[fixer] get workload failed.",
			"workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}

	newSpec, err := kw.fixPodSpec(dep.Spec.Template.Spec.DeepCopy())
	if err != nil {
		logs.Warnw("[fixer] modifyed failed.",
			"workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}
	dep.Spec.Template.Spec = *newSpec
	dep.Annotations["fix_log4j_at"] = time.Now().Format("2006-01-02T15:04:05")

	newDep, err := kw.kubeCluster.UpdateDeployment(dep)
	if err != nil {
		logs.Errorw("[fixer] update failed.", "workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}
	logs.Infof("[fixer] update succ. %v", newDep.Annotations)
	return nil
}

func (kw *kubeWorkload) fixDaemonSet() error {
	dep, err := kw.kubeCluster.GetDaemonSet(kw.wl.Namespace, kw.wl.Name)
	if err != nil {
		logs.Errorw("[fixer] get workload failed.",
			"workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}

	newSpec, err := kw.fixPodSpec(&dep.Spec.Template.Spec)
	if err != nil {
		logs.Warnw("[fixer] modifyed failed.",
			"workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}
	dep.Spec.Template.Spec = *newSpec
	dep.Annotations["fix_log4j_at"] = time.Now().Format("2006-01-02T15:04:05")

	newDep, err := kw.kubeCluster.UpdateDaemonSet(dep)
	if err != nil {
		logs.Errorw("[fixer] update failed.", "workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}
	logs.Infof("[fixer] update succ. %v", newDep.Annotations)
	return nil
}

func (kw *kubeWorkload) fixStatefulSet() error {
	dep, err := kw.kubeCluster.GetStatefulSet(kw.wl.Namespace, kw.wl.Name)
	if err != nil {
		logs.Errorw("[fixer] get workload failed.",
			"workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}

	newSpec, err := kw.fixPodSpec(dep.Spec.Template.Spec.DeepCopy())
	if err != nil {
		logs.Warnw("[fixer] modifyed failed.",
			"workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}
	dep.Spec.Template.Spec = *newSpec
	dep.Annotations["fix_log4j_at"] = time.Now().Format("2006-01-02T15:04:05")

	_, err = kw.kubeCluster.UpdateStatefulSet(dep)
	if err != nil {
		logs.Errorw("[fixer] update failed.", "workload", kw.wl.ToString(),
			"error", err,
		)
		return err
	}
	logs.Infow("[fixer] update succ.", "workload", kw.wl.ToString())
	return nil
}
func (kw *kubeWorkload) getPodContainer(name string) v1.Container {
	cons := make([]string, 0, len(kw.pod.Spec.Containers))
	for _, con := range kw.pod.Spec.Containers {
		cons = append(cons, con.Name)
		if con.Name == name {
			return con
		}
	}
	logs.Warnf("[fixer] not found container (%s) in %+v", name, cons)
	return v1.Container{}
}

func (kw *kubeWorkload) fixPodSpec(podspec *v1.PodSpec) (*v1.PodSpec, error) {
	modified := false
CONTAINER_LOOP:
	for i, con := range podspec.Containers {
		podCon := kw.getPodContainer(con.Name)
		if podCon.Name == "" {
			logs.Warnw("[fixer] not found container",
				"container", con.Name,
				"workload", kw.wl.ToString(),
			)
			continue
		}
		logs.Infow("[fixer] start modify spec",
			"container", con.Name,
			"command", podCon.Command,
			"args", podCon.Args,
			"workload", kw.wl.ToString(),
		)
		exiEnv := false
		for _, env := range con.Env {
			if env.Name == "FORMAT_MESSAGES_PATTERN_DISABLE_LOOKUPS" {
				exiEnv = true
				break
			}
		}
		if exiEnv {
			con.Env = append(con.Env, v1.EnvVar{
				Name:  "FORMAT_MESSAGES_PATTERN_DISABLE_LOOKUPS",
				Value: "true",
			})
			logs.Infow("[fixer] add env",
				"env", "FORMAT_MESSAGES_PATTERN_DISABLE_LOOKUPS",
				"container", con.Name,
				"workload", kw.wl.ToString(),
			)
			modified = true
		}
		for _, v := range append(podCon.Command, podCon.Args...) {
			if v == "-Dlog4j2.formatMsgNoLookups=true" {
				logs.Infow("[fixer] had fixed, skip",
					"container", con.Name,
					"workload", kw.wl.ToString(),
				)
				continue CONTAINER_LOOP
			}
		}
		if len(podCon.Args) > 0 && podCon.Args[0] == "java" {
			con.Args = insertAt(podCon.Args, "-Dlog4j2.formatMsgNoLookups=true", 1)
			modified = true
			podspec.Containers[i] = con
			logs.Infow("[fixer] modify args",
				"old", podCon.Args,
				"new", con.Args,
				"container", con.Name,
				"workload", kw.wl.ToString(),
			)
			continue
		}
		if len(podCon.Command) == 0 {
			logs.Warnw("[fixer] required container command",
				"command", con.Name,
				"workload", kw.wl.ToString(),
			)
			continue
		}
		if podCon.Command[0] != "java" {
			logs.Warnw("[fixer] not use java start container",
				"command", podCon.Command,
				"workload", kw.wl.ToString(),
			)
			continue
		}

		if len(podCon.Command) > 1 {
			con.Command = insertAt(podCon.Command, "-Dlog4j2.formatMsgNoLookups=true", 1)
			logs.Infow("[fixer] modify command",
				"old", podCon.Command,
				"new", con.Command,
				"container", con.Name,
				"workload", kw.wl.ToString(),
			)
		} else if len(podCon.Args) > 0 {
			con.Args = insertAt(podCon.Args, "-Dlog4j2.formatMsgNoLookups=true", 0)
			logs.Infow("[fixer] modify args",
				"old", podCon.Args,
				"new", con.Args,
				"container", con.Name,
				"workload", kw.wl.ToString(),
			)
		}
		modified = true
		podspec.Containers[i] = con
	}

	if !modified {
		return podspec, fmt.Errorf("[fixer] no modify")
	}
	return podspec, nil
}

func insertAt(sli []string, in string, idx int) []string {
	if idx > len(sli) {
		return append(sli, in)
	}
	sli = append(sli[:idx], append([]string{in}, sli[idx:]...)...)
	return sli
}

func isImageEqual(a, b string) bool {
	if a == b {
		return true
	}
	return false
}

// Image .
type Image struct {
	ContainerName string
	ImageID       string
	Image         string
}

// Tag image tag
func getImageNameTag(image string) (path string, tag string) {
	paths := strings.Split(image, "/")
	lastName := paths[len(paths)-1]
	names := strings.Split(lastName, ":")
	if len(names) == 1 {
		tag = "latest"
	} else {
		tag = names[len(names)-1]
		if tag == "" {
			tag = "latest"
		}
	}

	return
}

// ImageName image tag
func getImageName(image string) string {
	paths := strings.Split(image, "/")
	if strings.Contains(paths[0], ".") {
		if len(paths) > 1 {
			paths = paths[1:]
		}
	}

	path := strings.Join(paths, "/")
	idx := strings.Index(path, ":")
	if idx <= 0 {
		return path
	}
	return path[:idx]
}

// IsOfficialImage 是否是官方镜像name
func (i Image) IsOfficialImage(name string) bool {
	imgName := strings.ToLower(name)
	if name == imgName {
		return true
	}
	if imgName == "library/"+name {
		return true
	}
	if imgName == "docker.io/library/"+name {
		return true
	}
	if imgName == "docker.io/"+name {
		return true
	}
	return false
}
