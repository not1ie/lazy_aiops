package cicd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&CICDPipeline{}, &CICDExecution{}, &cmdb.Credential{}); err != nil {
		t.Fatalf("migrate schema failed: %v", err)
	}
	return db
}

func newEncryptedCredential(t *testing.T, db *gorm.DB, secretKey string, cred cmdb.Credential) cmdb.Credential {
	t.Helper()
	if err := cmdb.EncryptCredentialFields(secretKey, &cred); err != nil {
		t.Fatalf("encrypt credential failed: %v", err)
	}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatalf("create credential failed: %v", err)
	}
	return cred
}

func TestApplyCredentialInjectsJenkinsAuth(t *testing.T) {
	db := newTestDB(t)
	secretKey := "test-secret-key"
	h := &CICDHandler{db: db, secretKey: secretKey}

	cred := newEncryptedCredential(t, db, secretKey, cmdb.Credential{
		Name:     "jenkins-prod",
		Type:     "password",
		Username: "ci-bot",
		Password: "token-123",
	})

	p := CICDPipeline{
		Provider:     "jenkins",
		CredentialID: cred.ID,
		JenkinsUser:  "old-user",
		JenkinsToken: "old-token",
	}
	if err := h.applyCredential(&p); err != nil {
		t.Fatalf("applyCredential failed: %v", err)
	}
	if p.JenkinsUser != "ci-bot" {
		t.Fatalf("expected jenkins user ci-bot, got %q", p.JenkinsUser)
	}
	if p.JenkinsToken != "token-123" {
		t.Fatalf("expected jenkins token token-123, got %q", p.JenkinsToken)
	}
}

func TestTriggerBuildUsesCredentialForJenkins(t *testing.T) {
	db := newTestDB(t)
	secretKey := "test-secret-key"
	h := &CICDHandler{db: db, secretKey: secretKey}

	cred := newEncryptedCredential(t, db, secretKey, cmdb.Credential{
		Name:     "jenkins-shared",
		Type:     "password",
		Username: "ci-shared-user",
		Password: "ci-shared-token",
	})

	var gotPath, gotQuery, gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotUser, gotPass, _ = r.BasicAuth()
		w.Header().Set("Location", srvURLWithQueueID(r))
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	pipeline := CICDPipeline{
		Name:         "test-jenkins",
		Provider:     "jenkins",
		CredentialID: cred.ID,
		JenkinsURL:   srv.URL,
		JenkinsJob:   "demo-job",
	}

	exec, err := h.triggerBuild(&pipeline, map[string]string{"env": "prod"}, "manual", "tester", "")
	if err != nil {
		t.Fatalf("triggerBuild failed: %v", err)
	}
	if exec == nil || exec.ID == "" {
		t.Fatalf("expected execution created")
	}
	if gotPath != "/job/demo-job/buildWithParameters" {
		t.Fatalf("unexpected jenkins path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "env=prod") {
		t.Fatalf("missing jenkins query env=prod, got %q", gotQuery)
	}
	if gotUser != "ci-shared-user" || gotPass != "ci-shared-token" {
		t.Fatalf("expected basic auth from unified credential, got user=%q pass=%q", gotUser, gotPass)
	}

	var stored CICDExecution
	if err := db.First(&stored, "id = ?", exec.ID).Error; err != nil {
		t.Fatalf("execution not stored: %v", err)
	}
	if stored.RemoteBuildID == "" {
		t.Fatalf("expected remote build id populated")
	}
}

func srvURLWithQueueID(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/queue/item/1/"
}

func TestListCredentialsDoesNotExposeSecretFields(t *testing.T) {
	db := newTestDB(t)
	secretKey := "test-secret-key"
	h := &CICDHandler{db: db, secretKey: secretKey}

	_ = newEncryptedCredential(t, db, secretKey, cmdb.Credential{
		Name:       "gitlab-robot",
		Type:       "password",
		Username:   "bot",
		Password:   "secret-value",
		AccessKey:  "ak",
		SecretKey:  "sk",
		Passphrase: "pp",
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/cicd/credentials", h.ListCredentials)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cicd/credentials", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int                    `json:"code"`
		Data []CICDCredentialOption `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode json failed: %v", err)
	}
	if resp.Code != 0 || len(resp.Data) != 1 {
		t.Fatalf("unexpected response payload: %+v", resp)
	}

	var raw struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw json failed: %v", err)
	}
	if len(raw.Data) != 1 {
		t.Fatalf("unexpected raw payload: %s", w.Body.String())
	}
	for _, forbidden := range []string{"password", "secret_key", "private_key", "passphrase", "access_key"} {
		if _, exists := raw.Data[0][forbidden]; exists {
			t.Fatalf("response should not include key %s, body=%s", forbidden, w.Body.String())
		}
	}
}
