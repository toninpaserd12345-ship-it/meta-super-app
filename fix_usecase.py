import sys
content = open("backend/internal/usecase/automation.go").read()

content = content.replace('if pageID == "" || triggerType == "" || triggerValue == "" || productID == "" || replySetID == "" {\n\t\treturn nil, fmt.Errorf("%w: pageId, triggerType, triggerValue, productId, and replySetId are required", ErrInvalidAutomationRule)\n\t}', 'if pageID == "" || triggerType == "" || triggerValue == "" || replySetID == "" {\n\t\treturn nil, fmt.Errorf("%w: pageId, triggerType, triggerValue, and replySetId are required", ErrInvalidAutomationRule)\n\t}')

content = content.replace('if triggerType == "" || triggerValue == "" || productID == "" || replySetID == "" {\n\t\treturn fmt.Errorf("%w: triggerType, triggerValue, productId, and replySetId are required", ErrInvalidAutomationRule)\n\t}', 'if triggerType == "" || triggerValue == "" || replySetID == "" {\n\t\treturn fmt.Errorf("%w: triggerType, triggerValue, and replySetId are required", ErrInvalidAutomationRule)\n\t}')

open("backend/internal/usecase/automation.go", "w").write(content)
