package controller

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	log "github.com/win5do/go-lib/logx"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
)

const (
	PortClientName = "client"
	portClient     = 2379
	portPeer       = 2380
)

var (
	sccUser int64 = 1000
)

type ResourceBuilder struct {
	cr *dbv1.Etcd
}

func NewResourceBuilder(cr *dbv1.Etcd) *ResourceBuilder {
	return &ResourceBuilder{
		cr: cr,
	}
}

func (s *ResourceBuilder) StatefulSet(labels map[string]string) *appv1.StatefulSet {
	cr := s.cr

	name := cr.Name

	dataVolumeName := "data"

	replicas := int32(cr.Spec.Members)

	obj := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: cr.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            etcd,
							Image:           cr.Spec.Image,
							ImagePullPolicy: cr.Spec.ImagePullPolicy,
							Env:             s.Env(),
							Resources: corev1.ResourceRequirements{
								Limits:   s.resourceQuota(cr.Spec.Cpu, cr.Spec.Memory),
								Requests: s.resourceQuota(cr.Spec.Cpu, cr.Spec.Memory),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dataVolumeName,
									MountPath: "/var/run/etcd",
								},
							},
							ReadinessProbe: s.probe(portClient, 0, 10, 10, 3),
							LivenessProbe:  s.probe(portClient, 180, 10, 30, 10),
							Command:        s.command(),
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &sccUser,
					},
					ImagePullSecrets:   cr.Spec.ImagePullSecrets,
					ServiceAccountName: cr.Spec.ServiceAccountName,
					HostAliases:        cr.Spec.PodSpec.HostAliases,
					RestartPolicy:      cr.Spec.PodSpec.RestartPolicy,
					NodeSelector:       cr.Spec.PodSpec.NodeSelector,
					Affinity:           s.affinity(),
					Tolerations:        cr.Spec.PodSpec.Tolerations,
				},
			},
		},
	}

	volumes := make([]corev1.Volume, 0)

	// 是否需要数据持久化
	storage := cr.Spec.Storage
	if storage == "" {
		// emptyDir for test
		emptyDir := s.EmptyDirVolume(dataVolumeName)
		volumes = append(volumes, emptyDir)
	} else {
		// pvc.name ==  obj.name
		vo := corev1.Volume{
			Name: dataVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: dataVolumeName,
				},
			},
		}

		volumes = append(volumes, vo)

		obj.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			*s.PVC(dataVolumeName, storage),
		}
	}

	obj.Spec.Template.Spec.Volumes = volumes

	obj.Annotations = map[string]string{
		SpecHash: hashStr(obj.Spec),
	}

	return obj
}

// Ref: https://github.com/rustudorcalin/deploying-etcd-cluster
const etcdCmdTpl = `
SERVICE=%s
PEERS="%s"
exec etcd --name ${HOSTNAME} \
--listen-client-urls http://0.0.0.0:2379 \
--listen-peer-urls http://0.0.0.0:2380 \
--advertise-client-urls http://${HOSTNAME}.${SERVICE}:2379 \
--initial-advertise-peer-urls http://${HOSTNAME}.${SERVICE}:2380 \
--initial-cluster-token ${SERVICE} \
--initial-cluster ${PEERS} \
--initial-cluster-state new \
--data-dir /var/run/etcd/default.etcd
`

func (s *ResourceBuilder) command() []string {
	return []string{
		"sh",
		"-c",
		fmt.Sprintf(etcdCmdTpl, s.cr.Name, innerAddr(s.cr)),
	}
}

func (s *ResourceBuilder) resourceQuota(cpu, memory string) corev1.ResourceList {
	cr := s.cr

	mp := corev1.ResourceList{}

	if cr.Spec.Cpu != "" {
		mp[corev1.ResourceCPU] = resource.MustParse(cpu)
	}

	if cr.Spec.Memory != "" {
		mp[corev1.ResourceMemory] = resource.MustParse(memory)
	}

	return mp
}

func (s *ResourceBuilder) Env() []corev1.EnvVar {

	staticEnv := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	fields := []string{
		dbv1.DEBUG,
	}

	envVar := s.setEnvVar(fields)

	return append(staticEnv, envVar...)
}

func (s *ResourceBuilder) setEnvVar(fields []string) []corev1.EnvVar {
	var r []corev1.EnvVar

	for _, field := range fields {
		val := dbv1.GetEnv(s.cr.Spec.Env, field)
		if val == "" || val == dbv1.Empty {
			continue
		}

		r = append(r, corev1.EnvVar{
			Name:  field,
			Value: val,
		})
	}
	return r
}

func (s *ResourceBuilder) HeadlessService(name string, labels, selector map[string]string) *corev1.Service {
	cr := s.cr

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     portClient,
					Name:     PortClientName,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Port:     portPeer,
					Name:     "peer",
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector:  selector,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func (s *ResourceBuilder) ExportService(name string, labels, selector map[string]string) *corev1.Service {
	cr := s.cr

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     portClient,
					Name:     PortClientName,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: selector,
			Type:     corev1.ServiceTypeNodePort,
		},
	}

	return svc
}

func (s *ResourceBuilder) EmptyDirVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func (s *ResourceBuilder) PVC(name, storage string) *corev1.PersistentVolumeClaim {
	cr := s.cr

	var storageClassName *string
	if cr.Spec.StorageClassName != "" {
		log.Debugf("PVC storageClassName: %s", cr.Spec.StorageClassName)
		storageClassName = &cr.Spec.StorageClassName
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    baseLabel(cr.ObjectMeta),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storage),
				},
			},
		},
	}
}

func (s *ResourceBuilder) probe(port int, init, timeout, period, failure int32) *corev1.Probe {
	r := &corev1.Probe{}

	r.TCPSocket = &corev1.TCPSocketAction{
		Port: intstr.FromInt(port),
	}

	r.InitialDelaySeconds = init
	r.TimeoutSeconds = timeout
	r.PeriodSeconds = period
	r.FailureThreshold = failure

	return r
}

func (s *ResourceBuilder) affinity() *corev1.Affinity {
	if s.cr.Spec.PodSpec.Affinity != nil {
		return s.cr.Spec.PodSpec.Affinity
	}

	// 反亲和性配置，让pod最好不要分配到同一node上
	podAntiAffinity := &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 50,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: MemberLabel(s.cr.ObjectMeta, SelectAll),
					},
					TopologyKey: "kubernetes.io/hostname", // 此node label key默认存在
				},
			},
		},
	}

	return &corev1.Affinity{
		PodAntiAffinity: podAntiAffinity,
	}
}

func innerAddr(cr *dbv1.Etcd) string {
	var r strings.Builder
	const peer = "${pod-name}=http://${pod-name}.${svc-name}:2380"

	for i := 0; i < cr.Spec.Members; i++ {
		if i != 0 {
			r.WriteString(",")
		}
		svcName := cr.Name
		podName_ := podName(svcName, i)
		tmp := strings.ReplaceAll(peer, "${pod-name}", podName_)
		tmp = strings.ReplaceAll(tmp, "${svc-name}", svcName)
		r.WriteString(tmp)
	}

	return r.String()
}

func hashStr(data interface{}) string {
	hf := fnv.New32()

	bt, err := json.Marshal(data)
	if err != nil {
		log.Panic(err)
	}

	_, err = hf.Write(bt)
	if err != nil {
		log.Panic(err)
	}
	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}
