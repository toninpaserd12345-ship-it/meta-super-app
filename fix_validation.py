import sys
content = open("backend/internal/delivery/httpx/handler_automation.go").read()

old_validation = """	productID, replySetID = strings.TrimSpace(productID), strings.TrimSpace(replySetID)
	if productID == "" || replySetID == "" {
		return fmt.Errorf("productId and replySetId are required")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	replySet, err := h.Reply.GetSet(ctx, replySetID, accountID)
	if err != nil {
		return fmt.Errorf("the selected Reply Set does not belong to this account")
	}
	hasEnabledItem := false
	for _, item := range replySet.Items {
		if item.IsEnabled {
			hasEnabledItem = true
			break
		}
	}
	if !hasEnabledItem {
		return fmt.Errorf("the selected Reply Set needs at least one enabled message")
	}
	products, err := h.Product.ListProducts(ctx, userID(c), accountID)
	if err != nil {
		return fmt.Errorf("could not verify the selected Product: %w", err)
	}
	for _, product := range products {
		if product.ID == productID {
			return nil
		}
	}
	return fmt.Errorf("the selected Product does not belong to this account")"""

new_validation = """	productID, replySetID = strings.TrimSpace(productID), strings.TrimSpace(replySetID)
	if replySetID == "" {
		return fmt.Errorf("replySetId is required")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	replySet, err := h.Reply.GetSet(ctx, replySetID, accountID)
	if err != nil {
		return fmt.Errorf("the selected Reply Set does not belong to this account")
	}
	hasEnabledItem := false
	for _, item := range replySet.Items {
		if item.IsEnabled {
			hasEnabledItem = true
			break
		}
	}
	if !hasEnabledItem {
		return fmt.Errorf("the selected Reply Set needs at least one enabled message")
	}
	
	if productID != "" {
		products, err := h.Product.ListProducts(ctx, userID(c), accountID)
		if err != nil {
			return fmt.Errorf("could not verify the selected Product: %w", err)
		}
		for _, product := range products {
			if product.ID == productID {
				return nil
			}
		}
		return fmt.Errorf("the selected Product does not belong to this account")
	}
	return nil"""

content = content.replace(old_validation, new_validation)
open("backend/internal/delivery/httpx/handler_automation.go", "w").write(content)
