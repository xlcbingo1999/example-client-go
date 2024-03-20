package dynamicclient

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	Kubeconfig      string
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
)

const deploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.24
`

func listAllPods(config *restclient.Config) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	gvr := schema.GroupVersionResource{
		Group: "apps", Version: "v1", Resource: "deployments",
	}
	// 这里是可以根据 $group/$version/namespaces/$namespace/$resouce 去获取对应格式的资源情况
	// 例子: http://localhost:6443/apis/apps/v1/namespaces/default/deployments

	unstructObj, err := dynamicClient.Resource(gvr).Namespace("default").List(context.TODO(), metav1.ListOptions{Limit: 100})
	if err != nil {
		panic(err.Error())
	}

	podList := &apiv1.PodList{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.UnstructuredContent(), podList)
	if err != nil {
		panic(err.Error())
	}

	// 表头
	log.Printf("namespace\t status\t\t name\n")

	// 每个pod都打印namespace、status.Phase、name三个字段
	for _, d := range podList.Items {
		log.Printf("%v\t %v\t %v\n",
			d.Namespace,
			d.Status.Phase,
			d.Name)
	}
}

func createDeploymentBySSA(ctx context.Context, cfg *restclient.Config) error {
	// 构建一个restMapper用于寻找GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 构建了一个dynamic client
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	// 解析YAML到unstructured.Unstructured结构中
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(deploymentYAML), nil, obj)
	if err != nil {
		return err
	}

	// 寻找GVK
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// 从GVK中获取REST interface
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dyn.Resource(mapping.Resource)
	}

	// 将对象编组为json格式，最终应该是通过restful API去发?
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// 终于走到了创建资源的内容, 这里用的是Patch方法
	// sample-controller 是 kubernetes 官方提供的 CRD Controller 样例实现
	// 这里实现的方法可以快速得进行版本的切换, 是一个标准的声明式API接口!
	_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "sample-controller",
	})
	return err
}

func RunDynamicClient() {
	// home是家目录，如果能取得家目录的值，就可以用来做默认值
	if home := homedir.HomeDir(); home != "" {
		// 如果输入了kubeconfig参数，该参数的值就是kubeconfig文件的绝对路径，
		// 如果没有输入kubeconfig参数，就用默认路径~/.kube/config
		Kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", Kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// listAllPods(config)
	err = createDeploymentBySSA(context.TODO(), config)
	if err != nil {
		panic(err.Error())
	}

	listAllPods(config)
}
