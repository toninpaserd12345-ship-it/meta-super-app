import re

content = open("backend/internal/infrastructure/meta/graph_connector.go").read()
new_imports = """import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"gorm.io/gorm"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
"""
content = re.sub(r"import \(\n\t\"bytes\"\n\t\"context\"\n\t\"crypto/hmac\"\n\t\"crypto/rand\"\n\t\"crypto/sha256\"\n", new_imports, content)
content = content.replace("type GraphConnector struct {", "type GraphConnector struct {\n\tdb *gorm.DB\n")
content = content.replace("func NewGraphConnector(cfg GraphConfig, db *gorm.DB) *GraphConnector {", "func NewGraphConnector(cfg GraphConfig, db *gorm.DB) *GraphConnector {")
content = content.replace("g := &GraphConnector{cfg: cfg, client: &http.Client", "g := &GraphConnector{db: db, cfg: cfg, client: &http.Client")
open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
