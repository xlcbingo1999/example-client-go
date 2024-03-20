package informer

import (
	"log"
	"path/filepath"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	rest "k8s.io/client-go/rest"
)

var (
	Kubeconfig string
	config     *rest.Config
	err        error
)

func RunInformer() {
	// home是家目录，如果能取得家目录的值，就可以用来做默认值
	if home := homedir.HomeDir(); home != "" {
		// 如果输入了kubeconfig参数，该参数的值就是kubeconfig文件的绝对路径，
		// 如果没有输入kubeconfig参数，就用默认路径~/.kube/config
		Kubeconfig = filepath.Join(home, ".kube", "config")
	}

	if config, err = rest.InClusterConfig(); err != nil {
		if config, err = clientcmd.BuildConfigFromFlags("", Kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	// 创建 Clientset 对象
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// 初始化一个Informer Factory, 每隔30s就会重新List一次
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)
	// 对Deployment进行监听
	deployInformer := informerFactory.Apps().V1().Deployments()
	// 利用工厂模式进行Informer的创建
	informer := deployInformer.Informer()
	// 为Informer注册相关事件
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAddfunc,
		UpdateFunc: onUpdatefunc,
		DeleteFunc: onDelectfunc,
	})

	stopper := make(chan struct{})
	defer close(stopper)

	// 启动List and Watch 并等待所有启动的Informer的缓存被同步
	informerFactory.Start(stopper)
	informerFactory.WaitForCacheSync(stopper)

	// 创建一个Lister, 主要用于list资源使用
	deployLister := deployInformer.Lister()
	deployments, err := deployLister.Deployments("default").List(labels.Everything())
	if err != nil {
		panic(err.Error())
	}
	for idx, deploy := range deployments {
		log.Printf("%d -> %s\n", idx+1, deploy.Name)
	}
	<-stopper
}

func onAddfunc(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	log.Println("add a deployment: ", deploy.Name)
}

func onUpdatefunc(old, new interface{}) {
	oldDeploy := old.(*v1.Deployment)
	newDeploy := new.(*v1.Deployment)
	log.Println("update deployment: ", oldDeploy.Name, " ", newDeploy.Name)
}

func onDelectfunc(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	log.Println("delete a deployment: ", deploy.Name)
}
