package solver

import (
	"fmt"
	"sync"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/genchsusu/cert-manager-webhook-huawei/pkg/config"
	"github.com/genchsusu/cert-manager-webhook-huawei/pkg/huawei"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// huaweiDNSProviderSolver 实现了 cert-manager webhook 的 Solver 接口
type huaweiDNSProviderSolver struct {
	client     *kubernetes.Clientset
	dnsClients map[string]*huawei.Client
	sync.RWMutex
}

// NewSolver 创建一个新的 Huawei DNS Provider Solver 实例
func NewSolver() webhook.Solver {
	return &huaweiDNSProviderSolver{}
}

// Name 返回 solver 的名称
func (h *huaweiDNSProviderSolver) Name() string {
	return "huawei-dns"
}

// Present 用于处理 DNS-01 校验时创建 TXT 记录
func (h *huaweiDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Starting to add TXT record (value: %s): %v %v", ch.Key, ch.ResolvedFQDN, ch.ResolvedZone)

	cfg, err := config.LoadConfig(ch.Config)
	if err != nil {
		return err
	}
	klog.Infof("Parsed config: %+v", cfg)

	dnsClient, err := h.getDNSClient(cfg)
	if err != nil {
		klog.Errorf("Failed to get DNS client: %v", err)
		return err
	}

	// 查询现有记录
	record, err := dnsClient.GetTXTRecord(ch)
	if err != nil {
		return err
	}

	newValue := fmt.Sprintf("\"%s\"", ch.Key)

	// 如果记录不存在，创建记录
	if record == nil || record.Id == nil {
		err := dnsClient.CreateTXTRecord(ch, cfg.ZoneID)
		if err != nil {
			klog.Errorf("Failed to create TXT record: %v", err)
			return err
		}
		klog.Infof("TXT record created successfully: %v", ch.ResolvedFQDN)
		return nil
	}

	// 如果记录值已存在，跳过
	for _, val := range *record.Records {
		if val == newValue {
			klog.Infof("TXT record already exists, skipping: %v", ch.ResolvedFQDN)
			return nil
		}
	}

	// 否则追加值
	updatedRecords := append(*record.Records, newValue)
	err = dnsClient.UpdateTXTRecord(record, updatedRecords)
	if err != nil {
		klog.Errorf("Failed to update TXT record: %v", err)
		return err
	}

	klog.Infof("TXT record updated with new value: %v", ch.ResolvedFQDN)
	return nil
}

// CleanUp 用于在 DNS-01 校验完成后清理 TXT 记录
func (h *huaweiDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Starting to remove TXT record (value: %s): %v %v", ch.Key, ch.ResolvedFQDN, ch.ResolvedZone)

	cfg, err := config.LoadConfig(ch.Config)
	if err != nil {
		return err
	}
	klog.Infof("Parsed config: %+v", cfg)

	dnsClient, err := h.getDNSClient(cfg)
	if err != nil {
		klog.Errorf("Failed to get DNS client: %v", err)
		return err
	}

	// 获取记录
	record, err := dnsClient.GetTXTRecord(ch)
	if err != nil {
		return err
	}
	if record == nil || record.Id == nil {
		klog.Infof("No TXT record to delete for: %v", ch.ResolvedFQDN)
		return nil
	}

	// 构造新记录值（去掉当前的 key）
	target := fmt.Sprintf("\"%s\"", ch.Key)
	var remaining []string
	for _, val := range *record.Records {
		if val != target {
			remaining = append(remaining, val)
		}
	}

	// 如果为空，删除整个记录
	if len(remaining) == 0 {
		err := dnsClient.DeleteTXTRecord(record, ch)
		if err != nil {
			klog.Errorf("Failed to delete TXT record: %v", err)
			return err
		}
		klog.Infof("TXT record deleted completely: %v", ch.ResolvedFQDN)
		return nil
	}

	// 否则更新记录
	err = dnsClient.UpdateTXTRecord(record, remaining)
	if err != nil {
		klog.Errorf("Failed to update TXT record during cleanup: %v", err)
		return err
	}
	klog.Infof("TXT record value removed: %v", ch.ResolvedFQDN)
	return nil
}

// Initialize 初始化 Kubernetes 客户端和 DNS 客户端缓存
func (h *huaweiDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	h.client = cl
	h.dnsClients = make(map[string]*huawei.Client)
	return nil
}

// getDNSClient 根据 ZoneID 获取或创建一个 Huawei DNS 客户端
func (h *huaweiDNSProviderSolver) getDNSClient(cfg *config.HuaweiDNSProviderConfig) (*huawei.Client, error) {
	zoneID := cfg.ZoneID
	h.Lock()
	defer h.Unlock()

	if client, ok := h.dnsClients[zoneID]; ok {
		return client, nil
	}

	client, err := huawei.NewClient(cfg.AccessKey, cfg.SecretKey, cfg.Region)
	if err != nil {
		return nil, err
	}

	h.dnsClients[zoneID] = client
	return client, nil
}
