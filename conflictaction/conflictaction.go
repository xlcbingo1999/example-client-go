package conflictaction

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// deployment的名称
	DP_NAME string = "demo-deployment"
	// 用于更新的标签的名字
	LABEL_CUSTOMIZE string = "biz-version"
)

var (
	Kubeconfig string
)

func int32Ptr(i int32) *int32 {
	return &i
}

func create(clientset *kubernetes.Clientset) error {
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   DP_NAME,
			Labels: map[string]string{LABEL_CUSTOMIZE: "101"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	log.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	log.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

	return nil
}

func delete(clientset *kubernetes.Clientset, name string) error {
	deletePolicy := metav1.DeletePropagationBackground

	err := clientset.AppsV1().Deployments(apiv1.NamespaceDefault).Delete(context.TODO(), name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil {
		return err
	}

	log.Printf("Created deployment %s.\n", name)
	return nil
}

func get(clientset *kubernetes.Clientset, name string) (*appsv1.Deployment, error) {
	deployment, err := clientset.AppsV1().Deployments(apiv1.NamespaceDefault).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func updateByGetAndUpdate(clientset *kubernetes.Clientset, name string) error {
	deployment, err := clientset.AppsV1().Deployments(apiv1.NamespaceDefault).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	// 取出当前值
	currentVal, ok := deployment.Labels[LABEL_CUSTOMIZE]
	if !ok {
		return errors.New("未取得自定义标签")
	}

	// 将字符串类型转为int型
	val, err := strconv.Atoi(currentVal)

	if err != nil {
		log.Println("取得了无效的标签，重新赋初值")
		currentVal = "101"
	}

	// 将int型的label加一，再转为字符串
	deployment.Labels[LABEL_CUSTOMIZE] = strconv.Itoa(val + 1)

	_, err = clientset.AppsV1().Deployments(apiv1.NamespaceDefault).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	return err
}

type Confilct struct{}

func (conflict Confilct) DoAction(clientset *kubernetes.Clientset) error {
	log.Println("创建deployment")

	err := create(clientset)
	if err != nil {
		return err
	}

	<-time.NewTimer(1 * time.Second).C
	defer delete(clientset, DP_NAME)

	testNum := 5
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(testNum)

	startTime := time.Now().UnixMilli()

	for i := 0; i < testNum; i++ {
		go func(clientsetA *kubernetes.Clientset, index int) {
			defer waitGroup.Done()

			// err := updateByGetAndUpdate(clientsetA, DP_NAME) // 这里会出现并发冲突问题
			retryParam := wait.Backoff{
				Steps:    5, // 重试次数
				Duration: 10 * time.Millisecond,
				Factor:   1.0,
				Jitter:   0.1,
			}

			err := retry.RetryOnConflict(retryParam, func() error {
				return updateByGetAndUpdate(clientset, DP_NAME)
			})

			if err != nil {
				log.Printf("err: %v\n", err)
			}
		}(clientset, i)
	}

	waitGroup.Wait()

	// 再查一下，自定义标签的最终值
	deployment, err := get(clientset, DP_NAME)

	if err != nil {
		fmt.Printf("查询deployment发生异常: %v\n", err)
		return err
	}

	fmt.Printf("自定义标签的最终值为: %v，耗时%v毫秒\n", deployment.Labels[LABEL_CUSTOMIZE], time.Now().UnixMilli()-startTime)

	return nil
}

func RunConflictAction() {
	if home := homedir.HomeDir(); home != "" {
		Kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", Kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// 创建 ClientSet 实例
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	conflict := &Confilct{}
	err = conflict.DoAction(clientset)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	} else {
		fmt.Println("执行完成")
	}
}
