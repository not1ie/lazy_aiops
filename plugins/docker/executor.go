package docker

import (
	"fmt"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"gorm.io/gorm"
)

// GetExecutorByHost returns a command executor for a Docker host ID.
// It is shared by docker plugin handlers and cross-plugin integrations.
func GetExecutorByHost(db *gorm.DB, secretKey, dockerHostID string) (CommandExecutor, error) {
	var dHost DockerHost
	if err := db.First(&dHost, "id = ?", dockerHostID).Error; err != nil {
		return nil, fmt.Errorf("Docker主机不存在")
	}

	if dHost.HostID == "local" {
		return &LocalExecutor{}, nil
	}

	var host cmdb.Host
	if err := db.Preload("Credential").First(&host, "id = ?", dHost.HostID).Error; err != nil {
		return nil, fmt.Errorf("关联主机不存在")
	}
	if host.Credential == nil {
		return nil, fmt.Errorf("主机未配置凭据")
	}
	if err := cmdb.DecryptCredentialFields(secretKey, host.Credential); err != nil {
		return nil, fmt.Errorf("主机凭据解密失败")
	}

	return &core.SSHClient{
		Host:     host.IP,
		Port:     host.Port,
		Username: host.Credential.Username,
		Password: host.Credential.Password,
		Key:      host.Credential.PrivateKey,
		Timeout:  10 * time.Second,
	}, nil
}
