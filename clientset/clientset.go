package clientset

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/ptr"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NAMESPACE       = "test-clientset"
	DEPLOYMENT_NAME = "client-test-deployment"
	SERVICE_NAME    = "client-test-service"
)

var (
	Kubeconfig string
	Operate    string
)

func RunClientSet() {
	var err error
	var config *rest.Config

	if home := homedir.HomeDir(); home != "" {
		Kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		panic(err.Error())
	}

	flag.Parse()

	// 首先使用 inCluster 模式(需要去配置对应的 RBAC 权限，默认的sa是default->是没有获取deployments的List权限)
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 KubeConfig 文件创建集群配置 Config 对象
		if config, err = clientcmd.BuildConfigFromFlags("", Kubeconfig); err != nil {
			panic(err.Error())
		}
	}
	// 创建 ClientSet 实例
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("operation is %v\n", Operate)
	// 如果要执行清理操作
	if Operate == "clean" {
		clean(clientset)
	} else if Operate == "list" {
		listPod(clientset)
	} else {
		// 创建namespace
		createNamespace(clientset)

		// 创建deployment
		createDeployment(clientset)

		// 创建service
		createService(clientset)
	}

}

func clean(clientset *kubernetes.Clientset) {
	emptyDeleteOptions := metav1.DeleteOptions{}
	if err := clientset.CoreV1().Services(NAMESPACE).Delete(context.TODO(), SERVICE_NAME, emptyDeleteOptions); err != nil {
		panic(err.Error())
	}

	if err := clientset.AppsV1().Deployments(NAMESPACE).Delete(context.TODO(), DEPLOYMENT_NAME, emptyDeleteOptions); err != nil {
		panic(err.Error())
	}

	if err := clientset.CoreV1().Namespaces().Delete(context.TODO(), NAMESPACE, emptyDeleteOptions); err != nil {
		panic(err.Error())
	}
}

func createNamespace(clientset *kubernetes.Clientset) {
	namespaceClient := clientset.CoreV1().Namespaces()
	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NAMESPACE,
		},
	}

	result, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	log.Println("Create ns ", result.GetName())
}

func createService(clientset *kubernetes.Clientset) {
	serviceClient := clientset.CoreV1().Services(NAMESPACE)
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: SERVICE_NAME,
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:     "http",
				Port:     8080,
				NodePort: 30480,
			},
			},
			Selector: map[string]string{
				"app": "tomcat",
			},
			Type: apiv1.ServiceTypeNodePort,
		},
	}

	result, err := serviceClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	log.Println("Create SVC ", result.GetName())
}

func createDeployment(clientset *kubernetes.Clientset) {
	deploymentClient := clientset.AppsV1().Deployments(NAMESPACE)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: DEPLOYMENT_NAME,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(2)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "tomcat",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "tomcat",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{{
						Name:            "tomcat",
						Image:           "tomcat:8.0.18-jre8",
						ImagePullPolicy: "IfNotPresent",
						Ports: []apiv1.ContainerPort{{
							Name:          "http",
							Protocol:      apiv1.ProtocolSCTP,
							ContainerPort: 8080,
						},
						},
					},
					},
				},
			},
		},
	}

	result, err := deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	log.Println("Create deployment ", result.GetName())
}

func listPod(clientset *kubernetes.Clientset) {
	// 设置 list options
	listOptions := metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	}

	// 获取 default 命名空间下的 pod 列表
	pods, err := clientset.CoreV1().Pods(NAMESPACE).List(context.TODO(), listOptions)
	if err != nil {
		log.Fatal(err)
	}
	for _, pod := range pods.Items {
		log.Println(pod.Name)
	}
}
