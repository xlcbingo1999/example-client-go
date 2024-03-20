package controller

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

var (
	Kubeconfig string
	config     *rest.Config
	err        error
)

type Controller struct {
	indexer  cache.Indexer                   // 本地存储 负责存储完整资源信息的对象
	queue    workqueue.RateLimitingInterface // 业务逻辑的工作队列
	informer cache.Controller
}

func NewController(indexer cache.Indexer, queue workqueue.RateLimitingInterface, informer cache.Controller) *Controller {
	return &Controller{
		indexer:  indexer,
		informer: informer,
		queue:    queue,
	}
}

func (c *Controller) runWorker() {
	// 死循环 一直执行逻辑
	for c.processNextItem() {

	}
}

func (c *Controller) Run(workers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	defer c.queue.ShutDown() // 这个状态会被内部捕获到的
	klog.Info("Starting Pod Controller")

	go c.informer.Run(stopCh) // 开始接受从apiserver发出来的资源变更事件，并更新本地存储
	// 必须等到apiserver和本地存储实现同步才可以继续
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Time out waiting for cache to sync"))
		return
	}

	// 并发启动worker, 从工作队列里面拿数据, 然后执行业务逻辑
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping Pod Controller")
}

func (c *Controller) processNextItem() bool {
	// 阻塞等待直到有数据可以从工作队列里面被取出
	key, quit := c.queue.Get()

	// quit的情况是因为队列里面出现了外部的shutdown
	if quit {
		return false
	}

	// 将key从工作队列里面删除
	defer c.queue.Done(key)

	// 调用业务方法，实现具体的业务需求
	err := c.syncToStdout(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) syncToStdout(key string) error {
	// 根据key从本地存储中获取pod信息, 因为有长连接和apiserver保持同步, 因此本地的pod信息是和集群一致的
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		log.Printf("Pod %s does not exist anymore\n", key)
	} else {
		// 这里就是真正的业务逻辑代码了，一般会比较spce和status的差异，然后做出处理使得status与spce保持一致，
		// 此处为了代码简单仅仅打印一行日志
		log.Printf("Sync/Add/Update for Pod %s\n", obj.(*v1.Pod).GetName())
	}
	return nil
}

func (c *Controller) handleErr(err error, key interface{}) {
	// 没有错误时的处理逻辑
	if err == nil {
		// 确认这个key已经被成功处理，在队列中彻底清理掉
		// 假设之前在处理该key的时候曾报错导致重新进入队列等待重试，那么也会因为这个Forget方法而不再被重试
		c.queue.Forget(key)
		return
	}

	// 代码走到这里表示前面执行业务逻辑的时候发生了错误，
	// 检查已经重试的次数，如果不操作5次就继续重试，这里可以根据实际需求定制
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v: %v", key, err)
		c.queue.AddRateLimited(key)
		return
	}

	// 如果重试超过了5次就彻底放弃了，也像执行成功那样调用Forget做彻底清理（否则就没完没了了）
	c.queue.Forget(key)
	// 向外部报告错误，走通用的错误处理流程
	runtime.HandleError(err)
	klog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

func RunController() {
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

	// 创建一个ListWatch对象, 指定监控的资源的pod和namespace为default
	podListWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		v1.NamespaceDefault,
		fields.Everything(), // 表示啥都要监控
	)

	// 创建一个WorkerQueue, 这是一个限速队列
	// 限速队列: 需要周期性遍历执行，执行完毕需要再次执行，执行失败需要延时再次执行
	queue := workqueue.NewRateLimitingQueue(
		workqueue.DefaultControllerRateLimiter(),
	)

	// 这里是创建一个Informer, 内部核心的业务逻辑是三个增删改函数
	indexer, informer := cache.NewIndexerInformer(
		podListWatcher,
		&v1.Pod{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					queue.Add(key)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				if err == nil {
					queue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					queue.Add(key)
				}
			},
		},
		cache.Indexers{},
	)

	// 创建Controller对象，将所需的三个变量对象传入
	controller := NewController(indexer, queue, informer)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	// 在协程中启动controller
	go controller.Run(1, stop)

	// Wait forever
	select {}
}
