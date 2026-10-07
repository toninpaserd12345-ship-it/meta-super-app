import re

content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

# Update findPage
findPage_new = """func (g *GraphConnector) findPage(pageID string) (string, graphPage, bool) {
	g.mu.RLock()
	for accountID, session := range g.sessions {
		if page, ok := session.Pages[pageID]; ok {
			g.mu.RUnlock()
			return accountID, page, true
		}
	}
	g.mu.RUnlock()

	if g.db != nil {
		var model database.MetaPageTokenModel
		if err := g.db.First(&model, "page_id = ?", pageID).Error; err == nil {
			page := graphPage{
				MetaPage: domain.MetaPage{
					ID:         model.PageID,
					Name:       model.Name,
					Category:   model.Category,
					PictureURL: model.PictureURL,
					Connected:  model.IsConnected,
					TokenReady: model.AccessToken != "",
				},
				AccessToken: model.AccessToken,
			}
			g.mu.Lock()
			if g.sessions[model.AccountID] == nil {
				g.sessions[model.AccountID] = &graphSession{Pages: make(map[string]graphPage)}
			}
			g.sessions[model.AccountID].Pages[model.PageID] = page
			g.mu.Unlock()
			return model.AccountID, page, true
		}
	}
	return "", graphPage{}, false
}"""
content = re.sub(r"func \(g \*GraphConnector\) findPage\(pageID string\) \(string, graphPage, bool\) \{.*?return \"\", graphPage\{\}, false\n\}", findPage_new, content, flags=re.DOTALL)

# Update userToken
userToken_new = """func (g *GraphConnector) userToken(accountID string) (string, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	if session != nil && session.UserToken != "" {
		g.mu.RUnlock()
		return session.UserToken, nil
	}
	g.mu.RUnlock()

	if g.db != nil {
		var model database.MetaConnectionModel
		if err := g.db.First(&model, "account_id = ?", accountID).Error; err == nil {
			g.mu.Lock()
			if g.sessions[accountID] == nil {
				g.sessions[accountID] = &graphSession{Pages: make(map[string]graphPage)}
			}
			g.sessions[accountID].UserToken = model.UserToken
			g.mu.Unlock()
			return model.UserToken, nil
		}
	}
	return "", errors.New("facebook connection not found")
}"""
content = re.sub(r"func \(g \*GraphConnector\) userToken\(accountID string\) \(string, error\) \{.*?return \"\", errors\.New\(\"facebook connection not found\"\)\n\}", userToken_new, content, flags=re.DOTALL)

# Update OAuth success (in processOAuth)
# Right after: g.sessions[stateData.AccountID] = session
processOAuth_patch = """g.sessions[stateData.AccountID] = session
	if g.db != nil {
		g.db.Save(&database.MetaConnectionModel{AccountID: stateData.AccountID, UserToken: token.AccessToken})
	}"""
content = content.replace("g.sessions[stateData.AccountID] = session", processOAuth_patch)

# Update ConnectPage
# Right after: session.Pages[pageID] = page
connectPage_patch = """session.Pages[pageID] = page
	if g.db != nil {
		g.db.Save(&database.MetaPageTokenModel{
			PageID:      pageID,
			AccountID:   accountID,
			Name:        page.Name,
			Category:    page.Category,
			PictureURL:  page.PictureURL,
			AccessToken: page.AccessToken,
			IsConnected: true,
		})
	}"""
content = content.replace("session.Pages[pageID] = page", connectPage_patch)

# Update DisconnectPage
# Right after: session.Pages[pageID] = page
# Wait, let's find the exact DisconnectPage logic
disconnectPage_patch = """session.Pages[pageID] = page
	if g.db != nil {
		g.db.Model(&database.MetaPageTokenModel{}).Where("page_id = ?", pageID).Update("is_connected", false)
	}"""
# In DisconnectPage: it has `session.Pages[pageID] = page` as well!
# Wait, let's just do a manual replace for DisconnectPage to be safe.
