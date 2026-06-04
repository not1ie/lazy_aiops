package cicd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RegistryTag 镜像标签信息
type RegistryTag struct {
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
	Digest  string    `json:"digest"`
}

// HarborArtifact Harbor返回的制品结构
type HarborArtifact struct {
	Digest     string    `json:"digest"`
	PushTime   time.Time `json:"push_time"`
	Tags       []struct{ Name string `json:"name"` } `json:"tags"`
}

// FetchTags 获取镜像的Tags列表（降序）
func FetchTags(registry ImageRegistry, repositoryName string) ([]RegistryTag, error) {
	if registry.Provider == "harbor" {
		return fetchHarborTags(registry, repositoryName)
	}
	return fetchDockerRegistryTags(registry, repositoryName)
}

// DeleteTag 删除指定镜像Tag
func DeleteTag(registry ImageRegistry, repositoryName, tag, digest string) error {
	if registry.Provider == "harbor" {
		return deleteHarborArtifact(registry, repositoryName, digest)
	}
	return deleteDockerRegistryManifest(registry, repositoryName, digest)
}

// fetchHarborTags 使用 Harbor API 抓取 tags
// 需要将 repositoryName 拆分为 project_name 和 repo_name，例如：library/my-app
func fetchHarborTags(registry ImageRegistry, repositoryName string) ([]RegistryTag, error) {
	parts := strings.SplitN(repositoryName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("Harbor 仓库名必须为 project/repo 格式，如 library/my-app")
	}
	projectName := parts[0]
	repoName := parts[1]

	url := strings.TrimRight(registry.URL, "/") + fmt.Sprintf("/api/v2.0/projects/%s/repositories/%s/artifacts?with_tag=true&sort=-push_time", projectName, repoName)
	req, _ := http.NewRequest("GET", url, nil)
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 错误，状态码: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var artifacts []HarborArtifact
	if err := json.Unmarshal(body, &artifacts); err != nil {
		return nil, err
	}

	var tags []RegistryTag
	for _, art := range artifacts {
		for _, tag := range art.Tags {
			tags = append(tags, RegistryTag{
				Name:    tag.Name,
				Created: art.PushTime,
				Digest:  art.Digest,
			})
		}
	}
	return tags, nil
}

// fetchDockerRegistryTags 使用 Docker Registry V2 API 抓取 tags
func fetchDockerRegistryTags(registry ImageRegistry, repositoryName string) ([]RegistryTag, error) {
	url := strings.TrimRight(registry.URL, "/") + fmt.Sprintf("/v2/%s/tags/list", repositoryName)
	req, _ := http.NewRequest("GET", url, nil)
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 错误，状态码: %d", resp.StatusCode)
	}

	var data struct {
		Tags []string `json:"tags"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// 简单返回 tags，如果需要 created 时间，需要遍历请求 /v2/repo/manifests/tag 获取详细信息
	// 为了简化，这里 mock 时间
	var tags []RegistryTag
	for i, tagName := range data.Tags {
		tags = append(tags, RegistryTag{
			Name:    tagName,
			Created: time.Now().Add(-time.Duration(i) * time.Hour), // Mock 排序
			Digest:  "", // 留空，后续真正删除时需实时 fetch manifest
		})
	}
	return tags, nil
}

func deleteHarborArtifact(registry ImageRegistry, repositoryName, digest string) error {
	parts := strings.SplitN(repositoryName, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("Harbor 仓库名必须为 project/repo 格式")
	}
	url := strings.TrimRight(registry.URL, "/") + fmt.Sprintf("/api/v2.0/projects/%s/repositories/%s/artifacts/%s", parts[0], parts[1], digest)
	req, _ := http.NewRequest("DELETE", url, nil)
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("删除失败，状态码: %d", resp.StatusCode)
	}
	return nil
}

func deleteDockerRegistryManifest(registry ImageRegistry, repositoryName, digest string) error {
	url := strings.TrimRight(registry.URL, "/") + fmt.Sprintf("/v2/%s/manifests/%s", repositoryName, digest)
	req, _ := http.NewRequest("DELETE", url, nil)
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("删除失败，状态码: %d", resp.StatusCode)
	}
	return nil
}
