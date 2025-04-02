package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/genchsusu/cert-manager-webhook-huawei/pkg/solver"
	"k8s.io/klog/v2"
)

func main() {
	groupName := os.Getenv("GROUP_NAME")
	if groupName == "" {
		klog.Fatal("GROUP_NAME must be specified")
	}
	// 启动 webhook 服务，并传入 Solver 实现
	cmd.RunWebhookServer(groupName, solver.NewSolver())
}
