package huawei

import (
	"fmt"
	"strings"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
	region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
	"github.com/pkg/errors"
)

var RecordTypeTXT = "TXT"

// Client 封装了华为云 DNS 的客户端
type Client struct {
	dc *dns.DnsClient
}

// NewClient 创建一个新的华为云 DNS 客户端实例
func NewClient(ak, sk, regionName string) (*Client, error) {
	auth, err := basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("构建凭据失败: %w", err)
	}
	region, err := region.SafeValueOf(regionName)
	if err != nil {
		return nil, fmt.Errorf("无效的区域名称: %w", err)
	}
	builder := dns.DnsClientBuilder().
		WithRegion(region).
		WithCredential(auth)
	client, err := builder.SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("构建 DNS 客户端失败: %w", err)
	}
	dnsClient := dns.NewDnsClient(client)
	return &Client{dc: dnsClient}, nil
}

// getTXTRecordsRequest 构造用于查询 TXT 记录的请求
func (c *Client) getTXTRecordsRequest(ch *v1alpha1.ChallengeRequest) *model.ListRecordSetsRequest {
	request := &model.ListRecordSetsRequest{}
	recordType := RecordTypeTXT
	request.Type = &recordType
	nameRequest := extractRecordSetName(ch.ResolvedFQDN, ch.ResolvedZone)
	request.Name = &nameRequest
	recordsRequest := fmt.Sprintf("\"%s\"", ch.Key)
	request.Records = &recordsRequest
	return request
}

// GetTXTRecord 查询符合 challenge 请求的 TXT 记录
func (c *Client) GetTXTRecord(ch *v1alpha1.ChallengeRequest) (*model.ListRecordSetsWithTags, error) {
	request := c.getTXTRecordsRequest(ch)
	response, err := c.dc.ListRecordSets(request)
	if err != nil {
		return nil, err
	}

	for _, record := range *response.Recordsets {
		for _, value := range *record.Records {
			if value == fmt.Sprintf("\"%s\"", ch.Key) {
				return &record, nil
			}
		}
	}
	return nil, errors.Errorf("找不到 TXT 记录: %v", request.Name)
}

// CreateTXTRecord 创建或更新 TXT 记录
// 如果记录已存在则检查是否需要合并新值；否则创建一条新记录
func (c *Client) CreateTXTRecord(ch *v1alpha1.ChallengeRequest, zoneID string) error {
	// 查询现有记录
	request := c.getTXTRecordsRequest(ch)
	response, err := c.dc.ListRecordSets(request)
	if err != nil {
		return err
	}

	name := extractRecordSetName(ch.ResolvedFQDN, ch.ResolvedZone)
	newValue := fmt.Sprintf("\"%s\"", ch.Key)

	// 如果记录已存在，则合并 TXT 值
	if response != nil && len(*response.Recordsets) > 0 {
		record := (*response.Recordsets)[0]
		existing := *record.Records
		for _, r := range existing {
			if r == newValue {
				// 值已存在，无需更新
				return nil
			}
		}
		// 合并新值
		updated := append(existing, newValue)
		updateReq := &model.UpdateRecordSetRequest{
			ZoneId:      *record.ZoneId,
			RecordsetId: *record.Id,
			Body: &model.UpdateRecordSetReq{
				Name:    &name,
				Type:    &RecordTypeTXT,
				Records: &updated,
			},
		}
		_, err := c.dc.UpdateRecordSet(updateReq)
		return err
	}

	// 不存在记录，直接创建
	createReq := &model.CreateRecordSetRequest{
		ZoneId: zoneID,
		Body: &model.CreateRecordSetRequestBody{
			Name:    name,
			Type:    RecordTypeTXT,
			Records: []string{newValue},
		},
	}
	_, err = c.dc.CreateRecordSet(createReq)
	return err
}

// UpdateTXTRecord 更新指定记录的值
func (c *Client) UpdateTXTRecord(record *model.ListRecordSetsWithTags, records []string) error {
	req := &model.UpdateRecordSetRequest{
		ZoneId:      *record.ZoneId,
		RecordsetId: *record.Id,
		Body: &model.UpdateRecordSetReq{
			Name:    record.Name,
			Type:    &RecordTypeTXT,
			Records: &records,
		},
	}
	_, err := c.dc.UpdateRecordSet(req)
	return err
}

// DeleteTXTRecord 删除 TXT 记录中的指定值
// 如果记录中仅有该值，则删除整个记录；否则仅更新删除该值
func (c *Client) DeleteTXTRecord(record *model.ListRecordSetsWithTags, ch *v1alpha1.ChallengeRequest) error {
	if record.Id == nil || record.ZoneId == nil {
		return errors.New("待删除的记录无效")
	}

	targetValue := fmt.Sprintf("\"%s\"", ch.Key)
	var remaining []string
	for _, r := range *record.Records {
		if r != targetValue {
			remaining = append(remaining, r)
		}
	}

	if len(remaining) == 0 {
		// 无其它值，删除整条记录
		req := &model.DeleteRecordSetRequest{
			ZoneId:      *record.ZoneId,
			RecordsetId: *record.Id,
		}
		_, err := c.dc.DeleteRecordSet(req)
		if err != nil {
			return errors.Errorf("删除 TXT 记录失败: %v", err)
		}
		return nil
	}

	// 更新记录，删除目标值
	updateReq := &model.UpdateRecordSetRequest{
		ZoneId:      *record.ZoneId,
		RecordsetId: *record.Id,
		Body: &model.UpdateRecordSetReq{
			Name:    record.Name,
			Type:    &RecordTypeTXT,
			Records: &remaining,
		},
	}
	_, err := c.dc.UpdateRecordSet(updateReq)
	if err != nil {
		return errors.Errorf("局部删除 TXT 记录后更新失败: %v", err)
	}
	return nil
}

// extractRecordSetName 从 FQDN 中提取记录集名称
func extractRecordSetName(fqdn, zone string) string {
	name := util.UnFqdn(fqdn)
	if idx := strings.Index(name, "."+zone); idx != -1 {
		return name[:idx]
	}
	return name
}
