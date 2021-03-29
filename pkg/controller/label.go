package controller

import (
	"fmt"
	"strconv"

	log "github.com/win5do/go-lib/logx"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	labelRole = "role"
	etcd      = "etcd"
	Export    = "export"
	SelectAll = -999
	SpecHash  = "etcd-operator/spec-hash"
)

// cr的所有资源都打上这个label
func baseLabel(meta metav1.ObjectMeta) map[string]string {
	return map[string]string{
		"cr-name": meta.Name,
		"cr-uid":  string(meta.UID), // 使用uid，比name更安全
		labelRole: etcd,
	}
}

func exportLabel() map[string]string {
	return map[string]string{
		"svc": Export,
	}
}

func seqLabel(id int) map[string]string {

	if id == SelectAll {
		return nil
	}

	return map[string]string{
		"seq-id": strconv.Itoa(id),
	}
}

func podSelector(stsName string, id int) map[string]string {
	if id == SelectAll {
		return nil
	}

	return map[string]string{
		"statefulset.kubernetes.io/pod-name": podName(stsName, id),
	}
}

func podName(stsName string, id int) string {
	return fmt.Sprintf("%s-%d", stsName, id)
}

func ExportSvcLabel(meta metav1.ObjectMeta, id int) map[string]string {
	r := MergeLabels(baseLabel(meta), exportLabel(), seqLabel(id))
	return r
}

func MemberLabel(meta metav1.ObjectMeta, id int) map[string]string {
	r := MergeLabels(baseLabel(meta), podSelector(meta.Name, id))
	return r
}

// MergeLabels merges all the label maps received as argument into a single new label map.
func MergeLabels(allLabels ...map[string]string) map[string]string {
	res := map[string]string{}

	for _, labels := range allLabels {
		if labels == nil {
			continue
		}

		for k, v := range labels {
			if _, ok := res[k]; ok {
				log.Debugf("override label key: %s", k)
			}

			res[k] = v
		}
	}
	return res
}

func AddSuffix(a string, arr ...string) string {
	for _, s := range arr {
		a += "-" + s
	}
	return a
}
