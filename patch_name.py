import os
import glob

# 1. domain/automation.go
path = "backend/internal/domain/automation.go"
c = open(path).read()
c = c.replace("TriggerType  string    `json:\"triggerType\"`", "TriggerType  string    `json:\"triggerType\"`\n\tTriggerName  string    `json:\"triggerName\"`")
open(path, "w").write(c)

# 2. database/models.go
path = "backend/internal/infrastructure/database/models.go"
c = open(path).read()
c = c.replace("TriggerType  string        `gorm:\"size:50;not null\"` // post, ad, keyword", "TriggerType  string        `gorm:\"size:50;not null\"` // post, ad, keyword\n\tTriggerName  string        `gorm:\"size:255\"`")
open(path, "w").write(c)

# 3. repository/gorm_automation.go
path = "backend/internal/infrastructure/repository/gorm_automation.go"
c = open(path).read()
c = c.replace("TriggerType:  rule.TriggerType,\n\t\tTriggerValue: rule.TriggerValue,", "TriggerType:  rule.TriggerType,\n\t\tTriggerName:  rule.TriggerName,\n\t\tTriggerValue: rule.TriggerValue,")
c = c.replace("\"trigger_type\":  rule.TriggerType,\n\t\t\"trigger_value\": rule.TriggerValue,", "\"trigger_type\":  rule.TriggerType,\n\t\t\"trigger_name\":  rule.TriggerName,\n\t\t\"trigger_value\": rule.TriggerValue,")
c = c.replace("TriggerType:  m.TriggerType,\n\t\tTriggerValue: m.TriggerValue,", "TriggerType:  m.TriggerType,\n\t\tTriggerName:  m.TriggerName,\n\t\tTriggerValue: m.TriggerValue,")
open(path, "w").write(c)

# 4. usecase/automation.go
path = "backend/internal/usecase/automation.go"
c = open(path).read()
c = c.replace("func (u *Automation) CreateRule(ctx context.Context, accountID, pageID, triggerType, triggerValue, productID, replySetID string) (*domain.AutomationRule, error) {", "func (u *Automation) CreateRule(ctx context.Context, accountID, pageID, triggerType, triggerValue, triggerName, productID, replySetID string) (*domain.AutomationRule, error) {")
c = c.replace("triggerValue = strings.TrimSpace(triggerValue)", "triggerValue = strings.TrimSpace(triggerValue)\n\ttriggerName = strings.TrimSpace(triggerName)")
c = c.replace("TriggerValue: triggerValue,", "TriggerValue: triggerValue,\n\t\tTriggerName:  triggerName,")
c = c.replace("func (u *Automation) UpdateRule(ctx context.Context, id, accountID, triggerType, triggerValue, productID, replySetID string, isActive bool) error {", "func (u *Automation) UpdateRule(ctx context.Context, id, accountID, triggerType, triggerValue, triggerName, productID, replySetID string, isActive bool) error {")
c = c.replace("TriggerType:  triggerType,\n\t\tTriggerValue: triggerValue,", "TriggerType:  triggerType,\n\t\tTriggerName:  triggerName,\n\t\tTriggerValue: triggerValue,")
open(path, "w").write(c)

# 5. usecase/automation_test.go
path = "backend/internal/usecase/automation_test.go"
c = open(path).read()
c = c.replace("triggerValue string", "triggerValue string\n\t\ttriggerName  string")
c = c.replace("tt.triggerValue, tt.productID, tt.replySetID)", "tt.triggerValue, \"\", tt.productID, tt.replySetID)")
c = c.replace("triggerType+\"-value\", \"product-1\", \"reply-1\")", "triggerType+\"-value\", \"\", \"product-1\", \"reply-1\")")
c = c.replace("rule.TriggerValue, \"product-2\", \"reply-2\", false)", "rule.TriggerValue, \"\", \"product-2\", \"reply-2\", false)")
open(path, "w").write(c)

# 6. httpx/handler_automation.go
path = "backend/internal/delivery/httpx/handler_automation.go"
c = open(path).read()
c = c.replace("TriggerType  string `json:\"triggerType\"`\n\tTriggerValue string `json:\"triggerValue\"`", "TriggerType  string `json:\"triggerType\"`\n\tTriggerValue string `json:\"triggerValue\"`\n\tTriggerName  string `json:\"triggerName\"`")
c = c.replace("req.TriggerType, req.TriggerValue, req.ProductID", "req.TriggerType, req.TriggerValue, req.TriggerName, req.ProductID")
open(path, "w").write(c)
